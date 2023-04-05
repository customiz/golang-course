package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var Db *sql.DB

const basePath = "/api"
const productPath = "products"

type Product struct {
	ProductID    int     `json: "productid"`
	ProductName  string  `json:"productname"`
	ProductBrand string  `json:"productbrand"`
	Price        float64 `json: "price"`
}

//สร้าง API เชื่อมต่อกับฐานข้อมูล MySQL
func ConnectDB() {
	var err error
	Db, err = sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/go-course")

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Database Connected \n", Db)
	Db.SetConnMaxLifetime(time.Minute * 3)
	Db.SetMaxOpenConns(10)
	Db.SetMaxIdleConns(10)
}

func getProductList() ([]Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := Db.QueryContext(ctx, `SELECT * FROM product`)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	defer result.Close()
	products := make([]Product, 0)
	for result.Next() {
		var product Product
		result.Scan(&product.ProductID, &product.ProductName, &product.ProductBrand, &product.Price)

		products = append(products, product)
	}
	return products, nil
}

func insertProduct(product Product) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := Db.ExecContext(ctx, `INSERT INTO product (productname,productbrand,price) VALUES (?,?,?)`, product.ProductName, product.ProductBrand, product.Price)

	if err != nil {
		log.Println(err.Error())
		return 0, err
	}
	insertID, err := result.LastInsertId()
	if err != nil {
		log.Println(err.Error())
		return 0, err
	}
	return int(insertID), nil

}

func getProduct(productid int) (*Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	row := Db.QueryRowContext(ctx, `SELECT * FROM product WHERE productid = ?`, productid)

	product := &Product{}
	err := row.Scan(
		&product.ProductID,
		&product.ProductName,
		&product.ProductBrand,
		&product.Price,
	)
	if err != nil {
		log.Println(err)
		return nil, err
	} else if err == sql.ErrNoRows {
		return nil, nil
	}
	return product, nil

}

func handleProducts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	//เรียกดูข้อมูลทั้งหมด
	case http.MethodGet:
		productList, err := getProductList()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		j, err := json.Marshal(productList)
		if err != nil {
			log.Fatal(err)
		}
		_, err = w.Write(j)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Get all products from product ", productList)

	case http.MethodPost:
		var product Product
		err := json.NewDecoder(r.Body).Decode(&product)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ProductID, err := insertProduct(product)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(fmt.Sprintf(`{"productid":%d}`, ProductID)))
		fmt.Println("Product created")
	case http.MethodOptions:
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Println("Method Not Allowed")
	}

}

func handleProduct(w http.ResponseWriter, r *http.Request) {
	urlPathSegments := strings.Split(r.URL.Path, fmt.Sprintf("%s/", productPath))
	if len(urlPathSegments[1:]) > 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	productID, err := strconv.Atoi(urlPathSegments[len(urlPathSegments)-1])
	if err != nil {
		log.Print(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	switch r.Method {
	//เรียกดูเฉพาะ ID
	case http.MethodGet:
		product, err := getProduct(productID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if product == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		j, err := json.Marshal(product)
		if err != nil {
			log.Print(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, err = w.Write(j)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Get product ", product)

	}
}

//Middleware และ CORS
func corsMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization,X-CSRF-Token")
		handler.ServeHTTP(w, r)
	})

}

func SetupRoutes(apiBasePath string) {
	productsHandler := http.HandlerFunc(handleProducts)
	http.Handle(fmt.Sprintf("%s/%s", apiBasePath, productPath), corsMiddleware(productsHandler))

	productHandler := http.HandlerFunc(handleProduct)
	http.Handle(fmt.Sprintf("%s/%s/", apiBasePath, productPath), corsMiddleware(productHandler))
}

func main() {
	ConnectDB()
	SetupRoutes(basePath)
	log.Fatal(http.ListenAndServe(":5000", nil))

}
