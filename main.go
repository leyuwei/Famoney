package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
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
	Balances         map[string]float64
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
	OperatorID  int
	Operator    string
}

var db *sql.DB

var currencyRates = map[string]float64{}

func updateCurrencyRates() {
	exrate_api := os.Getenv("EXRATE_API")
	if exrate_api == "" {
		log.Fatal("EXRATE_API must be set")
	}
	resp, err := http.Get("https://v6.exchangerate-api.com/v6/" + exrate_api + "/latest/USD")
	if err != nil {
		log.Println("failed to fetch currency rates", err)
		return
	}
	defer resp.Body.Close()
	var data struct {
		Rates map[string]float64 `json:"conversion_rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Println("failed to decode currency rates", err)
		return
	}
	currencyRates = map[string]float64{"USD": 1}
	for k, v := range data.Rates {
		currencyRates[k] = v
	}
}

func currencyList() []string {
	codes := make([]string, 0, len(currencyRates))
	for k := range currencyRates {
		codes = append(codes, k)
	}
	sort.Strings(codes)
	return codes
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
		"Login":          "Login",
		"Register":       "Register",
		"Username":       "Username",
		"Password":       "Password",
		"Dashboard":      "Dashboard",
		"CreateWallet":   "Create Wallet",
		"WalletName":     "Wallet Name",
		"Currency":       "Currency",
		"Balance":        "Balance",
		"Add":            "Add",
		"Logout":         "Logout",
		"Category":       "Category",
		"Amount":         "Amount",
		"Description":    "Description",
		"Submit":         "Submit",
		"Confirm":        "Confirm",
		"Edit":           "Edit",
		"Delete":         "Delete",
		"View":           "View",
		"Share":          "Share",
		"Time":           "Time",
		"Flows":          "Flows",
		"Operator":       "Operator",
		"NoFlows":        "No flows",
		"NoWallets":      "No wallets",
		"Actions":        "Actions",
		"AddCategory":    "Add Category",
		"UpdateBalance":  "Update Balance",
		"ShareWallet":    "Share Wallet",
		"EditWallet":     "Edit Wallet",
		"EditCategories": "Edit Categories",
		"Summary":        "Summary",
		"TotalBalance":   "Total Balance",
		"ByCurrency":     "By Currency",
		"ByCategory":     "By Category",
		"Close":          "Close",
	},
	"zh": {
		"Login":          "登录",
		"Register":       "注册",
		"Username":       "用户名",
		"Password":       "密码",
		"Dashboard":      "仪表盘",
		"CreateWallet":   "创建钱包",
		"WalletName":     "钱包名称",
		"Currency":       "货币",
		"Balance":        "余额",
		"Add":            "添加",
		"Logout":         "退出登录",
		"Category":       "类别",
		"Amount":         "金额",
		"Description":    "描述",
		"Submit":         "提交",
		"Confirm":        "确认",
		"Edit":           "编辑",
		"Delete":         "删除",
		"View":           "查看",
		"Share":          "分享",
		"Time":           "时间",
		"Flows":          "流水",
		"Operator":       "操作人",
		"NoFlows":        "无流水",
		"NoWallets":      "没有钱包",
		"Actions":        "操作",
		"AddCategory":    "添加类别",
		"UpdateBalance":  "更新余额",
		"ShareWallet":    "分享钱包",
		"EditWallet":     "编辑钱包",
		"EditCategories": "编辑类别",
		"Summary":        "汇总",
		"TotalBalance":   "总余额",
		"ByCurrency":     "按货币",
		"ByCategory":     "按类别",
		"Close":          "关闭",
	},
}

func getBaseCurrency(w http.ResponseWriter, r *http.Request) string {
	if base := r.FormValue("base"); base != "" {
		http.SetCookie(w, &http.Cookie{Name: "base", Value: base, Path: "/"})
		return base
	}
	if c, err := r.Cookie("base"); err == nil {
		return c.Value
	}
	return "CNY"
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
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	if user == "" || pass == "" {
		log.Fatal("DB_USER and DB_PASSWORD must be set")
	}
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "3306"
	}
	name := os.Getenv("DB_NAME")
	if name == "" {
		name = "famoney"
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, name)
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
	updateCurrencyRates()
	go func() {
		for {
			time.Sleep(12 * time.Hour)
			updateCurrencyRates()
		}
	}()
	mux := http.NewServeMux()
	mux.HandleFunc("/famoney/", loginHandler)
	mux.HandleFunc("/famoney/login", loginHandler)
	mux.HandleFunc("/famoney/register", registerHandler)
	mux.HandleFunc("/famoney/logout", logoutHandler)
	mux.HandleFunc("/famoney/dashboard", auth(dashboardHandler))
	mux.HandleFunc("/famoney/wallet/create", auth(createWalletHandler))
	mux.HandleFunc("/famoney/wallet/", auth(viewWalletHandler))
	mux.HandleFunc("/famoney/category/add", auth(addCategoryHandler))
	mux.HandleFunc("/famoney/category/update", auth(updateCategoryHandler))
	mux.HandleFunc("/famoney/flow/", auth(flowHandler))
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
	base := getBaseCurrency(w, r)
	funcs := template.FuncMap{
		"T":       func(key string) string { return T(lang, key) },
		"Convert": convert,
	}
	data["Lang"] = lang
	data["BaseCurrency"] = base
	if _, ok := data["Currencies"]; !ok {
		data["Currencies"] = currencyList()
	}
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
	base := getBaseCurrency(w, r)

	rows, err := db.Query("SELECT w.id, w.name FROM wallets w JOIN wallet_owners o ON w.id=o.wallet_id WHERE o.user_id=?", uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	userWallets := []*Wallet{}
	currencyTotals := map[string]float64{}
	walletIDs := []int{}
	for rows.Next() {
		w := &Wallet{Balances: map[string]float64{}}
		if err := rows.Scan(&w.ID, &w.Name); err == nil {
			balRows, _ := db.Query("SELECT currency, balance FROM wallet_balances WHERE wallet_id=?", w.ID)
			for balRows.Next() {
				var cur string
				var bal float64
				if err := balRows.Scan(&cur, &bal); err == nil {
					w.Balances[cur] = bal
					currencyTotals[cur] += bal
				}
			}
			userWallets = append(userWallets, w)
			walletIDs = append(walletIDs, w.ID)
		}
	}

	catRows, _ := db.Query("SELECT id, name FROM categories")
	categories := []*Category{}
	categoriesMap := map[int]*Category{}
	for catRows.Next() {
		c := &Category{}
		if err := catRows.Scan(&c.ID, &c.Name); err == nil {
			categories = append(categories, c)
			categoriesMap[c.ID] = c
		}
	}

	categoryTotals := map[int]float64{}
	if len(walletIDs) > 0 {
		placeholders := make([]string, len(walletIDs))
		args := make([]interface{}, len(walletIDs))
		for i, id := range walletIDs {
			placeholders[i] = "?"
			args[i] = id
		}
		q := fmt.Sprintf("SELECT category_id, SUM(amount), currency FROM flows WHERE wallet_id IN (%s) GROUP BY category_id, currency", strings.Join(placeholders, ","))
		rows2, _ := db.Query(q, args...)
		for rows2.Next() {
			var cid int
			var sum float64
			var cur string
			if err := rows2.Scan(&cid, &sum, &cur); err == nil {
				categoryTotals[cid] += convert(sum, cur, base)
			}
		}
	}

	data := map[string]interface{}{
		"Wallets":        userWallets,
		"Categories":     categories,
		"CategoriesMap":  categoriesMap,
		"Currencies":     currencyList(),
		"CurrencyTotals": currencyTotals,
		"CategoryTotals": categoryTotals,
	}
	render(w, r, "dashboard.html", data)
}

func createWalletHandler(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("session_id")
	uid := sessionsStore[cookie.Value]
	name := r.FormValue("name")
	currency := r.FormValue("currency")
	res, err := db.Exec("INSERT INTO wallets (name) VALUES (?)", name)
	if err == nil {
		wid, _ := res.LastInsertId()
		db.Exec("INSERT INTO wallet_owners (wallet_id, user_id) VALUES (?, ?)", wid, uid)
		db.Exec("INSERT INTO wallet_balances (wallet_id, currency, balance) VALUES (?, ?, 0)", wid, currency)
	}
	http.Redirect(w, r, "/famoney/dashboard", http.StatusSeeOther)
}

func viewWalletHandler(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("session_id")
	uid := sessionsStore[cookie.Value]
	base := getBaseCurrency(w, r)
	path := strings.TrimPrefix(r.URL.Path, "/famoney/wallet/")
	if strings.HasSuffix(path, "/delete") && r.Method == "POST" {
		idStr := strings.TrimSuffix(path, "/delete")
		id, _ := strconv.Atoi(idStr)
		var count int
		db.QueryRow("SELECT COUNT(*) FROM wallet_owners WHERE wallet_id=? AND user_id=?", id, uid).Scan(&count)
		if count == 0 {
			http.NotFound(w, r)
			return
		}
		db.Exec("DELETE FROM flows WHERE wallet_id=?", id)
		db.Exec("DELETE FROM wallet_balances WHERE wallet_id=?", id)
		db.Exec("DELETE FROM wallet_owners WHERE wallet_id=?", id)
		db.Exec("DELETE FROM wallets WHERE id=?", id)
		http.Redirect(w, r, "/famoney/dashboard", http.StatusSeeOther)
		return
	}
	id, _ := strconv.Atoi(path)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM wallet_owners WHERE wallet_id=? AND user_id=?", id, uid).Scan(&count)
	if count == 0 {
		http.NotFound(w, r)
		return
	}

	wallet := &Wallet{Balances: map[string]float64{}}
	err := db.QueryRow("SELECT id, name FROM wallets WHERE id=?", id).Scan(&wallet.ID, &wallet.Name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	balRows, _ := db.Query("SELECT currency, balance FROM wallet_balances WHERE wallet_id=?", wallet.ID)
	for balRows.Next() {
		var cur string
		var bal float64
		if err := balRows.Scan(&cur, &bal); err == nil {
			wallet.Balances[cur] = bal
		}
	}

	if r.Method == "POST" {
		action := r.FormValue("action")
		switch action {
		case "flow":
			amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
			categoryID, _ := strconv.Atoi(r.FormValue("category"))
			desc := r.FormValue("description")
			cur := r.FormValue("currency")
			db.Exec("INSERT INTO wallet_balances (wallet_id, currency, balance) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE balance=balance+VALUES(balance)", wallet.ID, cur, amount)
			db.Exec("INSERT INTO flows (wallet_id, amount, currency, category_id, description, created_at, operator_id) VALUES (?, ?, ?, ?, ?, ?, ?)", wallet.ID, amount, cur, categoryID, desc, time.Now(), uid)
		case "balance":
			amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
			categoryID, _ := strconv.Atoi(r.FormValue("category"))
			desc := r.FormValue("description")
			cur := r.FormValue("currency")
			var old float64
			db.QueryRow("SELECT balance FROM wallet_balances WHERE wallet_id=? AND currency=?", wallet.ID, cur).Scan(&old)
			diff := amount - old
			db.Exec("INSERT INTO wallet_balances (wallet_id, currency, balance) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE balance=VALUES(balance)", wallet.ID, cur, amount)
			db.Exec("INSERT INTO flows (wallet_id, amount, currency, category_id, description, created_at, operator_id) VALUES (?, ?, ?, ?, ?, ?, ?)", wallet.ID, diff, cur, categoryID, desc, time.Now(), uid)
		case "share":
			username := r.FormValue("username")
			var uid2 int
			if err := db.QueryRow("SELECT id FROM users WHERE username=?", username).Scan(&uid2); err == nil {
				db.Exec("INSERT IGNORE INTO wallet_owners (wallet_id, user_id) VALUES (?, ?)", wallet.ID, uid2)
			}
		case "rename":
			name := r.FormValue("name")
			db.Exec("UPDATE wallets SET name=? WHERE id=?", name, wallet.ID)
			wallet.Name = name
		}
		balRows, _ := db.Query("SELECT currency, balance FROM wallet_balances WHERE wallet_id=?", wallet.ID)
		wallet.Balances = map[string]float64{}
		for balRows.Next() {
			var cur string
			var bal float64
			if err := balRows.Scan(&cur, &bal); err == nil {
				wallet.Balances[cur] = bal
			}
		}
	}

	wallet.CategoryBalances = map[int]float64{}
	balRows2, _ := db.Query("SELECT category_id, SUM(amount), currency FROM flows WHERE wallet_id=? GROUP BY category_id, currency", wallet.ID)
	for balRows2.Next() {
		var cid int
		var sum float64
		var cur string
		if err := balRows2.Scan(&cid, &sum, &cur); err == nil {
			wallet.CategoryBalances[cid] += convert(sum, cur, base)
		}
	}

	flowRows, _ := db.Query("SELECT f.id, f.wallet_id, f.amount, f.currency, f.category_id, f.description, f.created_at, u.id, u.username FROM flows f LEFT JOIN users u ON f.operator_id=u.id WHERE f.wallet_id=? ORDER BY f.created_at DESC", wallet.ID)
	walletFlows := []*Flow{}
	for flowRows.Next() {
		f := &Flow{}
		if err := flowRows.Scan(&f.ID, &f.WalletID, &f.Amount, &f.Currency, &f.CategoryID, &f.Description, &f.CreatedAt, &f.OperatorID, &f.Operator); err == nil {
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
	userRows, _ := db.Query("SELECT username FROM users")
	users := []string{}
	for userRows.Next() {
		var u string
		if err := userRows.Scan(&u); err == nil {
			users = append(users, u)
		}
	}

	data := map[string]interface{}{
		"Wallet":     wallet,
		"Flows":      walletFlows,
		"Categories": categories,
		"Currencies": currencyList(),
		"Users":      users,
	}
	render(w, r, "wallet.html", data)
}

func flowHandler(w http.ResponseWriter, r *http.Request) {
	cookie, _ := r.Cookie("session_id")
	uid := sessionsStore[cookie.Value]
	path := strings.TrimPrefix(r.URL.Path, "/famoney/flow/")
	if strings.HasSuffix(path, "/delete") && r.Method == "POST" {
		idStr := strings.TrimSuffix(path, "/delete")
		id, _ := strconv.Atoi(idStr)
		var wid int
		var amount float64
		var cur string
		db.QueryRow("SELECT wallet_id, amount, currency FROM flows WHERE id=?", id).Scan(&wid, &amount, &cur)
		var count int
		db.QueryRow("SELECT COUNT(*) FROM wallet_owners WHERE wallet_id=? AND user_id=?", wid, uid).Scan(&count)
		if count == 0 {
			http.NotFound(w, r)
			return
		}
		db.Exec("UPDATE wallet_balances SET balance=balance-? WHERE wallet_id=? AND currency=?", amount, wid, cur)
		db.Exec("DELETE FROM flows WHERE id=?", id)
		http.Redirect(w, r, fmt.Sprintf("/famoney/wallet/%d", wid), http.StatusSeeOther)
		return
	}
	if strings.HasSuffix(path, "/edit") {
		idStr := strings.TrimSuffix(path, "/edit")
		id, _ := strconv.Atoi(idStr)
		f := &Flow{}
		db.QueryRow("SELECT wallet_id, amount, currency, category_id, description FROM flows WHERE id=?", id).Scan(&f.WalletID, &f.Amount, &f.Currency, &f.CategoryID, &f.Description)
		var count int
		db.QueryRow("SELECT COUNT(*) FROM wallet_owners WHERE wallet_id=? AND user_id=?", f.WalletID, uid).Scan(&count)
		if count == 0 {
			http.NotFound(w, r)
			return
		}
		if r.Method == "POST" {
			newAmount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
			newCurrency := r.FormValue("currency")
			newCategoryID, _ := strconv.Atoi(r.FormValue("category"))
			newDesc := r.FormValue("description")
			db.Exec("UPDATE wallet_balances SET balance=balance-? WHERE wallet_id=? AND currency=?", f.Amount, f.WalletID, f.Currency)
			db.Exec("INSERT INTO wallet_balances (wallet_id, currency, balance) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE balance=balance+VALUES(balance)", f.WalletID, newCurrency, newAmount)
			db.Exec("UPDATE flows SET amount=?, currency=?, category_id=?, description=?, operator_id=? WHERE id=?", newAmount, newCurrency, newCategoryID, newDesc, uid, id)
			http.Redirect(w, r, fmt.Sprintf("/famoney/wallet/%d", f.WalletID), http.StatusSeeOther)
			return
		}
		catRows, _ := db.Query("SELECT id, name FROM categories")
		categories := []*Category{}
		for catRows.Next() {
			c := &Category{}
			if err := catRows.Scan(&c.ID, &c.Name); err == nil {
				categories = append(categories, c)
			}
		}
		data := map[string]interface{}{
			"Flow":       f,
			"Categories": categories,
			"Currencies": currencyList(),
		}
		render(w, r, "flow_edit.html", data)
		return
	}
	http.NotFound(w, r)
}

func addCategoryHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if name != "" {
		db.Exec("INSERT INTO categories (name) VALUES (?) ON DUPLICATE KEY UPDATE name=name", name)
	}
	http.Redirect(w, r, "/famoney/dashboard", http.StatusSeeOther)
}

func updateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.FormValue("id")
	name := r.FormValue("name")
	if idStr != "" && name != "" {
		id, _ := strconv.Atoi(idStr)
		db.Exec("UPDATE categories SET name=? WHERE id=?", name, id)
	}
	http.Redirect(w, r, "/famoney/dashboard", http.StatusSeeOther)
}
