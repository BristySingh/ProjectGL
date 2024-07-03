package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Customer model
type Customer struct {
	CustomerId   int    `json:"customerid"`
	CustomerName string `json:"customername"`
}

// DB connection
var db *sql.DB

func main() {
	// Connect to the PostgreSQL database
	var err error
	db, err = sql.Open("postgres", `postgres://postgres:abcd1234@localhost/postgres?sslmode=disable`)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize router
	router := mux.NewRouter()

	// Define routes
	router.HandleFunc("/customer", getCustomers).Methods("GET")
	router.HandleFunc("/customer/{id}", getCustomerById).Methods("GET")
	router.HandleFunc("/customer", createCustomer).Methods("POST")
	router.HandleFunc("/customer/{id}", updateCustomer).Methods("PUT")
	router.HandleFunc("/customer/{id}", deleteCustomer).Methods("DELETE")

	// Start server
	log.Fatal(http.ListenAndServe(":9000", router))
}

// Get customers
func getCustomers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT customerid, customername FROM customer")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	customer := make([]Customer, 0)
	for rows.Next() {
		var c Customer
		if err := rows.Scan(&c.CustomerId, &c.CustomerName); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		customer = append(customer, c)
	}

	respondWithJSON(w, http.StatusOK, customer)
}

// Get customer by id
func getCustomerById(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	row := db.QueryRow("SELECT customerid, customername FROM customer WHERE customerid = $1", params["customerid"])
	var c Customer
	err := row.Scan(&c.CustomerId, &c.CustomerName)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, c)
}

// Create a new customer
func createCustomer(w http.ResponseWriter, r *http.Request) {
	var c Customer
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&c); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	err := db.QueryRow("INSERT INTO customer(customername) VALUES($1) RETURNING customerid", c.CustomerName).Scan(&c.CustomerId)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, c)
}

// Update an existing customer
func updateCustomer(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var c Customer
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := db.Exec("UPDATE customer SET customername = $1 WHERE customerid = $2", c.CustomerName, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	c.CustomerId = parseInt(id)
	json.NewEncoder(w).Encode(c)
}

func deleteCustomer(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	_, err := db.Exec("DELETE FROM customer WHERE customerid = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

// Helper functions

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
