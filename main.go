package main

import (
        "fmt"
        "net/http"
        "encoding/json"
        "time"
        "log"
        //"strings"
        "github.com/gorilla/mux"
        "github.com/dghubble/go-twitter/twitter"
        "github.com/dghubble/oauth1"
        "os"
        //"reflect"
)

type GoogleResponse struct {
        Kind string `json:"kind"`
        Url string `json:"url"`
        Items [] ItemJson `json:"items"`
}

type ItemJson struct {
        Snippet string `json:"snippet"`
        Title string `json:"title"`
        Link string `json:"link"`
}

type DuckDuckGoResponse struct {
        RelatedTopics [] RelatedTopicsJson `json:"RelatedTopics"`
}

type RelatedTopicsJson struct {
        FirstURL string `json:"FirstURL"`
        Text string `json:"Text"`
}

type Result struct {
    Url string `json:"url"`
    Text string `json:"text"`
}

type ResultError struct {
    Res Result
    Err error
}

type FinalResponse struct {
    Query string `json:"query"`
    Results struct {
        Google Result `json:"google"`
        Duckduckgo Result `json:"duckduckgo"`
        Twitter Result `json:"twitter"`
    } `json:"results"` 
} 


func (gr *GoogleResponse) searchApiResponse(url string, ch chan<-ResultError) {
    timeout := time.Duration(1 * time.Second)
    client := http.Client{Timeout: timeout}
    resp, err := client.Get(url)
    var resErr ResultError
    if err != nil {
        fmt.Println("Error when sending request", err)
        resErr.Res = Result{Url: "", Text: ""}
        resErr.Err = err
        // er <- err
        ch <- resErr
        return
    }
    defer resp.Body.Close()
    json.NewDecoder(resp.Body).Decode(gr)
    var res Result
    //res.Website = "google"
    if len(gr.Items) > 0 {
        res.Url = gr.Items[0].Link
        res.Text = gr.Items[0].Snippet
    } else {
        res.Text = ""
    }
    // res.Text = gr.Items[0].Snippet
    resErr.Res = res
    resErr.Err = nil
    ch <- resErr
    close(ch)
    // er <- err
}


// type DuckDuckGoResponse struct {
//         RelatedTopics [] RelatedTopicsJson `json:"RelatedTopics"`
// }

// type RelatedTopicsJson struct {
//         FirstURL string `json:"FirstURL"`
//         Text string `json:"Text"`
// }

func (dr *DuckDuckGoResponse) searchApiResponse(url string, ch chan<-ResultError) {
    var resErr ResultError
    fmt.Println(url)
    timeout := time.Duration(1 * time.Second)
    client := http.Client{Timeout: timeout}
    resp, err := client.Get(url)
    if err != nil {
        fmt.Println("Error when sending request", err)
        resErr.Res = Result{Url: "", Text: ""}
        resErr.Err = err
        // er <- err
        ch <- resErr
        return
    }
    defer resp.Body.Close()
    json.NewDecoder(resp.Body).Decode(dr)
    var res Result
    if len(dr.RelatedTopics) > 0 {
        res.Url = dr.RelatedTopics[0].FirstURL
        res.Text = dr.RelatedTopics[0].Text
    } else {
        res.Url = ""
        res.Text = ""
    }
    resErr.Res = res
    resErr.Err = nil
    ch <- resErr
    close(ch)
    // er <- err
}

func twitterApiResponse(query string, ch chan<-ResultError) {
    var resErr ResultError
    config := oauth1.NewConfig("mRCUJqJgBP9G855MO0FAcAp3o", "bYNVX1VokBClYistDWId4ybFNvtILOCAE5Hjh8lmArSxl5jol6")
    token := oauth1.NewToken("859847563012300800-choBYeORCnp2YEQV1gdjom2p26FM68v", "xmSMl4Gc68XTaMLXbSJWSKIGFabr6HgTWVcw5qPLzAamw")
    httpClient := config.Client(oauth1.NoContext, token)
    client := twitter.NewClient(httpClient)
    //timeline_endpoint = "https://api.twitter.com/1.1/search/tweets.json?q=Tomato"
    search, _, err := client.Search.Tweets(&twitter.SearchTweetParams{
        Query: query,
    })
    var res Result
    if err == nil {
        fmt.Println("in twitter if")
        if len(search.Statuses) > 0 {
            res.Url = "https://api.twitter.com/1.1/search/tweets.json?q="+query
            res.Text = search.Statuses[0].Text
        } else {
            res = Result{Url: "", Text: ""}
        }
    } else {
        fmt.Println("in twitter else")
        res = Result{Url: "", Text: ""}
    }
    fmt.Println(res.Text)
    resErr.Res = res
    resErr.Err = err
    ch <- resErr
    close(ch)
}

// type Result struct {
//     Url string `json:"url"`
//     Text string `json:"text"`
// }

// type ResultError struct {
//     Res Result
//     Err error
// }

// type FinalResponse struct {
//     Query string `json:"query"`
//     Results struct {
//         Google Result `json:"google"`
//         Duckduckgo Result `json:"duckduckgo"`
//         Twitter Result `json:"twitter"`
//     } `json:"results"` 
// } 

func getResult(ch <-chan ResultError) (string, string){
    var url string
    var text string
    select {
    case resErr := <-ch:
        if v:= resErr.Err; v == nil {
            url = resErr.Res.Url
            text = resErr.Res.Text
        } else {
            url = resErr.Res.Url
            text = v.Error()
        }
    case <-time.After(time.Second * 1):
        url = ""
        text = "Timeout"
    }
    return url, text
}

func search(writer http.ResponseWriter, request *http.Request) {
    q := request.URL.Query().Get("q")
    googleUrl := "https://www.googleapis.com/customsearch/v1?key=AIzaSyA4knzICz0O6nzS3Rx5tpeJgD5Sj3Ip0aM&cx=005606248206714305298:xatfsisnorq&q="+q
    duckDuckGoUrl := "http://api.duckduckgo.com/?format=json&q="+q
    var gr GoogleResponse
    var dr DuckDuckGoResponse
    goCh := make(chan ResultError)
    duCh := make(chan ResultError)
    twCh := make(chan ResultError)
    go gr.searchApiResponse(googleUrl, goCh)
    go dr.searchApiResponse(duckDuckGoUrl, duCh)
    go twitterApiResponse(q, twCh)
    var response FinalResponse
    response.Query = q
    response.Results.Google.Url, response.Results.Google.Text = getResult(goCh)
    response.Results.Duckduckgo.Url, response.Results.Duckduckgo.Text = getResult(duCh)
    response.Results.Twitter.Url, response.Results.Twitter.Text = getResult(twCh)
    res_str, _ := json.Marshal(response)
    // close(goCh)
    // close(duCh)
    // close(twCh)
    // return res_str
    writer.Write([]byte(res_str))        
}

func main() {
    r := mux.NewRouter()
    r.HandleFunc("/search", search).Methods("GET")
    log.Fatal(http.ListenAndServe(GetPort(), r))
    // res_str := callApis()
    // fmt.Println(string(res_str))
}

func GetPort() string {
     var port = os.Getenv("PORT")
     // Set a default port if there is nothing in the environment
     if port == "" {
         port = "4747"
         fmt.Println("INFO: No PORT environment variable detected, defaulting to " + port)
     }
     return ":" + port
}

