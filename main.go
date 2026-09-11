package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Subscriber struct {
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	Subscribed time.Time `json:"subscribed"`
}

const dataFile = "subscribers.json"

var (
	mu          sync.RWMutex
	subscribers []Subscriber
)

func loadSubscribers() {
	mu.Lock()
	defer mu.Unlock()
	b, err := os.ReadFile(dataFile)
	if err != nil {
		return
	}
	_ = json.Unmarshal(b, &subscribers)
}

func saveSubscribers() error {
	mu.RLock()
	defer mu.RUnlock()
	b, err := json.MarshalIndent(subscribers, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dataFile, b, 0o644)
}

func main() {
	loadSubscribers()

	mux := http.NewServeMux()
	mux.HandleFunc("/", servePage)
	mux.HandleFunc("/subscribe", handlerSubscribe)
	mux.HandleFunc("/subscribers", handlerSubscribers)

	addr := ":8080"
	log.Printf("News server running at http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func servePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.FileServer(http.Dir(".")).ServeHTTP(w, r)
}

func handlerSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	name := strings.TrimSpace(r.FormValue("name"))
	if email == "" || !strings.Contains(email, "@") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "a valid email is required"})
		return
	}

	mu.Lock()
	for _, s := range subscribers {
		if s.Email == email {
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "already subscribed"})
			return
		}
	}
	subscribers = append(subscribers, Subscriber{Email: email, Name: name, Subscribed: time.Now()})
	mu.Unlock()

	if err := saveSubscribers(); err != nil {
		log.Printf("save subscribers: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"ok": "subscribed"})
}

func handlerSubscribers(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	defer mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscribers)
}
