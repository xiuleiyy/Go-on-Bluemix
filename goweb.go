package main

import (
	"html/template"
	"net/http"
	"log"
	"os"
	"strconv"
	"fmt"
)

type Results struct {
  Title    string // Indicates success or failure
  Message  string // the result message
  Details  string // the details of the message
}

const (
  DEFAULT_PORT = "4001"

  primesHeader = `
<html>
  <head>
    <title>Go Sample App - {{.Title}}</title>
    <link rel="stylesheet" href="/stylesheets/style.css">
  </head>

  <body>

    <h1>{{.Title}}</h1>
    <p><div>{{.Message}}</div></p>
    <div>
`

  primesFooter = `
    </div>
    <p><div><a href="/">Back home</a></div></p>

  </body>
</html>
`
)

var resultsTemplate = template.Must(template.ParseFiles("templates/results.html"))
var primesHeaderTemplate = template.Must(template.New("header").Parse(primesHeader))
var primesFooterTemplate = template.Must(template.New("footer").Parse(primesFooter))


// Display the index html page
//
func publicHandler(w http.ResponseWriter, r *http.Request) {
  log.Println(">> publicHandler")
  defer log.Println("<< publicHandler")

  if r.Method != "GET" {
    w.WriteHeader(http.StatusMethodNotAllowed)
    return
  }

  path := "public" + r.URL.Path
  if path == "public/" {
    path = "public/index.html"
  }
  log.Printf("Serving file: %+v\n", path)
  http.ServeFile(w, r, path)
}


// Put in another file and make it a library
//
type InvalidArgument uint

func (n InvalidArgument) Error() string {
  return fmt.Sprintf("Invalid argument number %d", n)
}

type Factor struct {
  Num uint64
  Pow uint
}

// Return a sorted array of Factor, where each element is the
// prime factor with the associated power
//
func PrimeFactors(n uint64) ([]Factor, error) {
  if n < 2 {
    return nil, InvalidArgument(n)
  }

  factors := make([]Factor, 0)

  var i uint
  for i=0; n%2 == 0; i++ { n /= 2 }
  if i > 0 { factors = append(factors, Factor{Num: 2, Pow:i}) }

  for p:=uint64(3); (p*p) <= n; p+=2 {

    for i=0; n%p == 0; i++ { n /= p }
    if i > 0 { factors = append(factors, Factor{Num: p, Pow:i}) }

  }

  if n > 1 { factors = append(factors, Factor{Num: n, Pow:1}) }

  return factors, nil
}

// Calculate first n prime numbers and send them
// over a channel
//
func FirstNPrimeNumbers(n uint, ch chan uint64) (error) {
  log.Println(">> FirstNPrimeNumbers")
  defer log.Println("<< FirstNPrimeNumbers")

  if n < 3 {
    return InvalidArgument(n)
  }

  primes := make([]uint64, n)
  primes[0] = 2
  primes[1] = 3
  primes[2] = 5
  cprime := uint64(7)

  ch <- primes[0]
  ch <- primes[1]
  ch <- primes[2]

  for i:=uint(3); i<n; cprime+=2 {
    if (cprime%5) != 0 {
      isprime := true
      for p := 0; (primes[p]*primes[p]) <= cprime; p++ {
        if (cprime%primes[p]) == 0 {
          isprime = false
          break
        }
      }
      if isprime {
        primes[i] = cprime
        i++
        ch <- cprime
      }
    }
  }
  return nil
}

func prettyPrintPrimeFactors(n uint64, f []Factor) string {
  s := fmt.Sprintf("Prime Factors of %+v = ", n)
  for i, _ := range f {
    if i==0 {
      s += fmt.Sprintf("%v^%v", f[i].Num, f[i].Pow)
    } else {
      s += fmt.Sprintf(" * %v^%v", f[i].Num, f[i].Pow)
    }
  }
  return s
}



// Rest handler to get prime factors of a number
// GET /primefactors?number=[the number]
//
func primeFactorsHandler(w http.ResponseWriter, r *http.Request) {
  log.Println(">> primeFactorsHandler")
  defer log.Println("<< primeFactorsHandler")

  if r.Method != "POST" {
    w.WriteHeader(http.StatusMethodNotAllowed)
    return
  }

  num_s := r.FormValue("number")
  log.Printf("Got number from input form: %+v\n", num_s)
  var results *Results
  if num, err := strconv.Atoi(num_s); err==nil {
    // calculate prime factors
    if num > 0 {
      if factors, err := PrimeFactors(uint64(num)); err == nil {
        ss := prettyPrintPrimeFactors(uint64(num), factors)
        results = &Results{Title: "Success", Message: ss}
      } else {
        results = &Results{Title: "Error",
                           Message: fmt.Sprintf("Cannot calculate prime factors of %+v", num),
                           Details: err.Error()}
      }
    } else {
      results = &Results{Title: "Error",
                         Message: fmt.Sprintf("Cannot calculate prime factors of %+v", num),
                         Details: "It must be greather than zero."}
    }
  } else {
    results = &Results{Title: "Error",
                       Message: "Invalid number",
                       Details: err.Error()}
  }
  renderResults(w, results)
}

func primeNumbersHandler(w http.ResponseWriter, r *http.Request) {
  log.Println(">> primeNumbersHandler")
  defer log.Println("<< primeNumbersHandler")

  if r.Method != "POST" {
    w.WriteHeader(http.StatusMethodNotAllowed)
    return
  }

  num_s := r.FormValue("limit")
  log.Printf("Got limit from input form: %+v\n", num_s)
  var results *Results
  if num, err := strconv.Atoi(num_s); err==nil {
    // calculate first limit prime numbers
    if num > 0 {
      ch := make(chan uint64, 1)
      quit := make(chan *Results, 1)
      go func() {
        if err := FirstNPrimeNumbers(uint(num), ch); err != nil {
          results = &Results{Title: "Error",
                             Message: fmt.Sprintf("Cannot calculate first %+v prime numbers", num),
                             Details: err.Error()}
        }
        // signal the normal end of calculation
        quit <- results
      }()

      writeHead := true
      tab := 0
      for {
        select {
          case v := <- ch:
            if writeHead {
              // do write header
              primesHeaderTemplate.Execute(w, &Results{Title: "Success", Message: fmt.Sprintf("First %+v prime numbers:\n", num)})
              writeHead = false
            }
            fmt.Fprintf(w, "%+v  ", v)
            if tab++; tab > 14 {
              fmt.Fprintf(w, "<br>")
              tab = 0
            }

          case res := <- quit:
            if res == nil {
              primesFooterTemplate.Execute(w, nil)
            } else {
              renderResults(w, results)
            }
            return
        }
      }

    } else {
      results = &Results{Title: "Error",
                         Message: fmt.Sprintf("Cannot calculate first %+v prime numbers", num),
                         Details: "Limit must be greather than zero."}
    }
  } else {
    results = &Results{Title: "Error",
                       Message: "Invalid number",
                       Details: err.Error()}
  }
  if results != nil {
    log.Printf("Renedring results %+v\n", results) // debug only
    renderResults(w, results)
  }
}

func renderResults(w http.ResponseWriter, results *Results) {
  log.Println(">> renderResults")
  defer log.Println("<< renderResults")

  if err := resultsTemplate.ExecuteTemplate(w, "results.html", results); err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }
}

func main() {
  // Configure the standard logger
  log.SetFlags(log.Lmicroseconds|log.Ldate|log.Lshortfile)

  log.Println(">> GoWeb")
  defer log.Println("<< GoWeb")

  var port string
  if port = os.Getenv("VCAP_APP_PORT"); len(port)==0 {
    log.Printf("Warning, VCAP_APP_PORT not set. Defaulting to %+v\n", DEFAULT_PORT)
    port = DEFAULT_PORT
  }

  http.HandleFunc("/primefactors", primeFactorsHandler)
  http.HandleFunc("/primenumbers", primeNumbersHandler)
  http.HandleFunc("/", publicHandler)

  log.Printf("Starting GoWeb on port %+v\n", port)
  http.ListenAndServe(":" + port, nil)
}
