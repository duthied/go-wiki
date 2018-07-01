package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

type Page struct {
	Title string
	Body  []byte
}

type RootPage struct {
	Title string
	Pages []*Page
}

const DataPath = "data/"

// write Page
func (p *Page) save() error {
	filename := DataPath + p.Title + ".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

// read Page
func loadPage(title string) (*Page, error) {
	filename := DataPath + title + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func formatTitle(t string) string {
	i := strings.LastIndex(t, ".")
	n := t[0:i]
	return strings.ToLower(n)
}

func getPages() ([]*Page, error) {
	dList, err := ioutil.ReadDir(DataPath)
	if err != nil {
		return nil, err
	}

	var pages []*Page
	for _, file := range dList {
		title := formatTitle(file.Name())
		page, err := loadPage(title)
		if err != nil {
			return nil, err
		}
		pages = append(pages, page)
	}

	return pages, nil
}

var templates = make(map[string]*template.Template)

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates[tmpl].ExecuteTemplate(w, "layout", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("handling request:", r.URL)

		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	renderTemplate(w, "view", p)
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func renderRootTemplate(w http.ResponseWriter, tmpl string, p *RootPage) {
	err := templates[tmpl].ExecuteTemplate(w, "layout", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("handling request:", r.URL)

	pages, err := getPages()
	if err != nil {
		http.Error(w, "Unable to read list", 500)
		return
	}

	c := &RootPage{Title: "Home", Pages: pages}
	renderRootTemplate(w, "root", c)
}

func main() {

	templates["root"] = template.Must(template.ParseFiles("tmpl/root.html", "tmpl/layout.html"))
	templates["view"] = template.Must(template.ParseFiles("tmpl/view.html", "tmpl/layout.html"))
	templates["edit"] = template.Must(template.ParseFiles("tmpl/edit.html", "tmpl/layout.html"))

	fmt.Println("Serving ...")

	// register handlers
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))

	http.HandleFunc("/", rootHandler)

	http.ListenAndServe(":8080", nil)
}
