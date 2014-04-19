package main

import (
    "fmt"
    rss "github.com/jteeuwen/go-pkg-rss"
    "code.google.com/p/go.net/html"
    "os"
    "time"
    "strings"
    "log"
    "sort"
)

// The date go-pkg-rss parses dates as, not sure what it is
const rssDateFormat = "Mon, 2 Jan 2006 15:04:05 -0700"

// True while processing first load of an rss feed
var initialLoad bool = true 

// Represents an item from your dashboard feed
type HgRssItem struct {
    Item *rss.Item
    PubDate time.Time
    Commits []string
}

// The array of feed items
var items = make([]*HgRssItem,0,10)

// Struct for sorting feed items oldest->newest
type ByDate []*HgRssItem

// Implements the sort interface
func (a ByDate) Len() int {
    return len(a)
}
func (a ByDate) Swap(i, j int) {
    a[i], a[j] = a[j], a[i]
}
func (a ByDate) Less(i, j int) bool {
    return a[i].PubDate.Before(a[j].PubDate)
}

func main() {
    // Poll the feed every x minutes

    if len(os.Args) < 2 {
        fmt.Printf("Usage: hgfeed url-to-feed\n")
        os.Exit(1)
    }

    PollFeed(os.Args[1],1)
}


func PollFeed(uri string, timeout int) {
    feed := rss.New(timeout, true, chanHandler, itemHandler)

    for {
        if err := feed.Fetch(uri, nil); err != nil {
            fmt.Fprintf(os.Stderr, "[e] %s: %s", uri, err)
            return
        }

        <-time.After(time.Duration(feed.SecondsTillUpdate() * 1e9))
    }
}

// Called after the itemHandler has processed all the new items
func chanHandler(feed *rss.Feed, newchannels []*rss.Channel) {
    sort.Sort(ByDate(items))

    for _,item := range items {
        printItem(item)
    }

    initialLoad = false 
}

// Called when new items are found on the feed
func itemHandler(feed *rss.Feed, ch *rss.Channel, newitems []*rss.Item) {
    for _,item := range newitems {
        hgRssItem := createHgRssItem(item)

        if(initialLoad) {
            items = append(items, hgRssItem)
        } else {
            printItem(hgRssItem)
        }
    }
}

func printItem(item *HgRssItem) {
    fmt.Printf("\x1b[36;1m%s -> \x1b[30;1m%s -> \x1b[32;1m%s\n", item.Item.Author.Name, item.PubDate.Format("Jan 2, 2006 15:05PM"), item.Item.Title)

    for _, commit := range item.Commits {
        commitParts := strings.Split(commit, " - ")
        commit = commitParts[1]
        fmt.Printf("\t \x1b[35;1m-> \x1b[0m%s\n", commit)
    }

    fmt.Println()
}

// Parses commits from the <Description>
func createHgRssItem(item *rss.Item) *HgRssItem {
    doc, err := html.Parse(strings.NewReader(item.Description))
    if err != nil {
        log.Fatal(err)
    }
    
    commits := make([]string,0,10);

    var parseCommits func(*html.Node)
    parseCommits = func(n *html.Node) {
        if n.Type == html.ElementNode && n.Data == "li" {
            commits = append(commits, n.FirstChild.Data)
        }
        for c:= n.FirstChild; c != nil; c = c.NextSibling {
            parseCommits(c)
        }
    }

    parseCommits(doc)

    hgRssItem := new(HgRssItem)
    hgRssItem.Item = item
    hgRssItem.PubDate,_ = time.Parse(rssDateFormat, item.PubDate)
    hgRssItem.Commits = commits

    return hgRssItem
}