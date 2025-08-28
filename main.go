package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	_ "github.com/go-sql-driver/mysql"
)

// Data models

type User struct {
	ID       int
	Username string
	Password string // In production use hashed passwords
}

type Wallet struct {
	ID               int
	Name             string
	Currency         string
	Balance          float64
	Owners           []int
	CategoryBalances map[int]float64
}

type Category struct {
	ID   int
	Name string
}

type Flow struct {
	ID          int
	WalletID    int
	Amount      float64
	Currency    string
	CategoryID  int
	Description string
	CreatedAt   time.Time
}

var db *sql.DB

var currencyRates = map[string]float64{
	"USD": 1,
	"CNY": 0.14,
	"EUR": 1.1,
}

func convert(amount float64, from, to string) float64 {
	rateFrom, okFrom := currencyRates[from]
	rateTo, okTo := currencyRates[to]
	if !okFrom || !okTo {
		return amount
	}
	usd := amount * rateFrom
	return usd / rateTo
}

var translations = map[string]map[string]string{
	"en": {
		"Login":         "Login",
		"Register":      "Register",
		"Username":      "Username",
		"Password":      "Password",
		"Dashboard":     "Dashboard",
		"CreateWallet":  "Create Wallet",
		"WalletName":    "Wallet Name",
		"Currency":      "Currency",
		"Balance":       "Balance",
		"Add":           "Add",
		"Logout":        "Logout",
		"Category":      "Category",
		"Amount":        "Amount",
		"Description":   "Description",
		"Submit":        "Submit",
		"AddCategory":   "Add Category",
		"UpdateBalance": "Update Balance",
	},
	"zh": {
		"Login":         "登录",
		"Register":      "注册",
		"Username":      "用户名",
		"Password":      "密码",
		"Dashboard":     "仪表盘",
		"CreateWallet":  "创建钱包",
		"WalletName":    "钱包名称",
		"Currency":      "货币",
		"Balance":       "余额",
		"Add":           "添加",
		"Logout":        "退出登录",
		"Category":      "类别",
		"Amount":        "金额",
		"Description":   "描述",
		"Submit":        "提交",
		"AddCategory":   "添加类别",
		"UpdateBalance": "更新余额",
	},
}

var sessionsStore = map[string]int{}

func newSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}

func initDB() {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "user:password@tcp(127.0.0.1:3306)/famoney?parseTime=true"
	}
	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	initDB()
	mux := http.NewServeMux()
	mux.HandleFunc("/famoney/", loginHandler)
	mux.HandleFunc("/famoney/login", loginHandler)
	mux.HandleFunc("/famoney/register", registerHandler)
	mux.HandleFunc("/famoney/logout", logoutHandler)
	mux.HandleFunc("/famoney/dashboard", auth(dashboardHandler))
	mux.HandleFunc("/famoney/wallet/create", auth(createWalletHandler))
	mux.HandleFunc("/famoney/wallet/", auth(viewWalletHandler))
	mux.HandleFunc("/famoney/category/add", auth(addCategoryHandler))
	mux.Handle("/famoney/static/", http.StripPrefix("/famoney/static/", http.FileServer(http.Dir("static"))))

	log.Println("Server running on :8295")
	http.ListenAndServe(":8295", mux)
}

func auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Redirect(w, r, "/famoney/login", http.StatusSeeOther)
			return
		}
		if _, ok := sessionsStore[cookie.Value]; !ok {
			http.Redirect(w, r, "/famoney/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func getLang(w http.ResponseWriter, r *http.Request) string {
	if lang := r.FormValue("lang"); lang != "" {
		http.SetCookie(w, &http.Cookie{Name: "lang", Value: lang, Path: "/"})
		return lang
	}
	if c, err := r.Cookie("lang"); err == nil {
		return c.Value
	}
	return "zh"
}

func T(lang, key string) string {
	if v, ok := translations[lang][key]; ok {
		return v
	}
	return key
}

func render(w http.ResponseWriter, r *http.Request, tmpl string, data map[string]interface{}) {
	lang := getLang(w, r)
	funcs := template.FuncMap{
		"T":       func(key string) string { return T(lang, key) },
		"Convert": convert,
	}
	data["Lang"] = lang
	t, err := template.New("layout.html").Funcs(funcs).ParseFiles(filepath.Join("templates", "layout.html"), filepath.Join("templates", tmpl))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.ExecuteTemplate(w, "layout", data)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")
		var id int
		var dbPass string
		err := db.QueryRow("SELECT id, password FROM users WHERE username=?", username).Scan(&id, &dbPass)
		if err == nil && dbPass == password {
			sid := newSessionID()
			sessionsStore[sid] = id
			http.SetCookie(w, &http.Cookie{Name: "session_id", Value: sid, Path: "/"})
			http.Redirect(w, r, "/famoney/dashboard", http.StatusSeeOther)
			return
		}
	}
	render(w, r, "login.html", map[string]interface{}{})
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")
		if _, err := db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", username, password); err == nil {
			http.Redirect(w, r, "/famoney/login", http.StatusSeeOther)
			return
		}
	}
	render(w, r, "register.html", map[string]interface{}{})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("session_id"); err == nil {
		delete(sessionsStore, cookie.Value)
		http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "", Path: "/", MaxAge: -1})
	}
	http.Redirect(w, r, "/famoney/login", http.StatusSeeOther)
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("session_id")
	uid := sessionsStore[cookie.Value]

	rows, err := db.Query("SELECT w.id, w.name, w.currency, w.balance FROM wallets w JOIN wallet_owners o ON w.id=o.wallet_id WHERE o.user_id=?", uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	userWallets := []*Wallet{}
	for rows.Next() {
		w := &Wallet{}
		if err := rows.Scan(&w.ID, &w.Name, &w.Currency, &w.Balance); err == nil {
			userWallets = append(userWallets, w)
		}
	}

	catRows, _ := db.Query("SELECT id, name FROM categories")
	categories := map[int]*Category{}
	for catRows.Next() {
		c := &Category{}
		if err := catRows.Scan(&c.ID, &c.Name); err == nil {
			categories[c.ID] = c
		}
	}
	data := map[string]interface{}{
		"Wallets":    userWallets,
		"Categories": categories,
		"Rates":      currencyRates,
	}
	render(w, r, "dashboard.html", data)
}

func createWalletHandler(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("session_id")
	uid := sessionsStore[cookie.Value]
	name := r.FormValue("name")
	currency := r.FormValue("currency")
	res, err := db.Exec("INSERT INTO wallets (name, currency, balance) VALUES (?, ?, 0)", name, currency)
	if err == nil {
		wid, _ := res.LastInsertId()
		db.Exec("INSERT INTO wallet_owners (wallet_id, user_id) VALUES (?, ?)", wid, uid)
	}
	http.Redirect(w, r, "/famoney/dashboard", http.StatusSeeOther)
}

func viewWalletHandler(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("session_id")
	uid := sessionsStore[cookie.Value]

	idStr := strings.TrimPrefix(r.URL.Path, "/famoney/wallet/")
	id, _ := strconv.Atoi(idStr)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM wallet_owners WHERE wallet_id=? AND user_id=?", id, uid).Scan(&count)
	if count == 0 {
		http.NotFound(w, r)
		return
	}

	wallet := &Wallet{}
	err := db.QueryRow("SELECT id, name, currency, balance FROM wallets WHERE id=?", id).Scan(&wallet.ID, &wallet.Name, &wallet.Currency, &wallet.Balance)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if r.Method == "POST" {
		action := r.FormValue("action")
		amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
		categoryID, _ := strconv.Atoi(r.FormValue("category"))
		desc := r.FormValue("description")
		if action == "flow" {
			db.Exec("UPDATE wallets SET balance = balance + ? WHERE id=?", amount, wallet.ID)
			db.Exec("INSERT INTO flows (wallet_id, amount, currency, category_id, description, created_at) VALUES (?, ?, ?, ?, ?, ?)", wallet.ID, amount, wallet.Currency, categoryID, desc, time.Now())
		} else if action == "balance" {
			diff := amount - wallet.Balance
			db.Exec("UPDATE wallets SET balance = ? WHERE id=?", amount, wallet.ID)
			db.Exec("INSERT INTO flows (wallet_id, amount, currency, category_id, description, created_at) VALUES (?, ?, ?, ?, ?, ?)", wallet.ID, diff, wallet.Currency, categoryID, desc, time.Now())
		}
		db.QueryRow("SELECT balance FROM wallets WHERE id=?", wallet.ID).Scan(&wallet.Balance)
	}

	wallet.CategoryBalances = map[int]float64{}
	balRows, _ := db.Query("SELECT category_id, SUM(amount) FROM flows WHERE wallet_id=? GROUP BY category_id", wallet.ID)
	for balRows.Next() {
		var cid int
		var sum float64
		if err := balRows.Scan(&cid, &sum); err == nil {
			wallet.CategoryBalances[cid] = sum
		}
	}

	flowRows, _ := db.Query("SELECT id, wallet_id, amount, currency, category_id, description, created_at FROM flows WHERE wallet_id=? ORDER BY created_at DESC", wallet.ID)
	walletFlows := []*Flow{}
	for flowRows.Next() {
		f := &Flow{}
		if err := flowRows.Scan(&f.ID, &f.WalletID, &f.Amount, &f.Currency, &f.CategoryID, &f.Description, &f.CreatedAt); err == nil {
			walletFlows = append(walletFlows, f)
		}
	}

	catRows, _ := db.Query("SELECT id, name FROM categories")
	categories := map[int]*Category{}
	for catRows.Next() {
		c := &Category{}
		if err := catRows.Scan(&c.ID, &c.Name); err == nil {
			categories[c.ID] = c
		}
	}

	data := map[string]interface{}{
		"Wallet":     wallet,
		"Flows":      walletFlows,
		"Categories": categories,
	}
	render(w, r, "wallet.html", data)
}

func addCategoryHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if name != "" {
		db.Exec("INSERT INTO categories (name) VALUES (?) ON DUPLICATE KEY UPDATE name=name", name)
	}
	http.Redirect(w, r, "/famoney/dashboard", http.StatusSeeOther)
}
