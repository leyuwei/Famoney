package main

import (
	"crypto/rand"
	"encoding/hex"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
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

var (
	users      = map[int]*User{}
	wallets    = map[int]*Wallet{}
	categories = map[int]*Category{}
	flows      = map[int]*Flow{}

	nextUserID     = 1
	nextWalletID   = 1
	nextCategoryID = 1
	nextFlowID     = 1
)

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

func main() {
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
        "T": func(key string) string { return T(lang, key) },
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
		for _, u := range users {
			if u.Username == username && u.Password == password {
				sid := newSessionID()
				sessionsStore[sid] = u.ID
				http.SetCookie(w, &http.Cookie{Name: "session_id", Value: sid, Path: "/"})
				http.Redirect(w, r, "/famoney/dashboard", http.StatusSeeOther)
				return
			}
		}
	}
	render(w, r, "login.html", map[string]interface{}{})
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")
		u := &User{ID: nextUserID, Username: username, Password: password}
		users[nextUserID] = u
		nextUserID++
		http.Redirect(w, r, "/famoney/login", http.StatusSeeOther)
		return
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
	userWallets := []*Wallet{}
	for _, wallet := range wallets {
		for _, owner := range wallet.Owners {
			if owner == uid {
				userWallets = append(userWallets, wallet)
				break
			}
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
	wallet := &Wallet{ID: nextWalletID, Name: name, Currency: currency, Owners: []int{uid}, CategoryBalances: map[int]float64{}}
	wallets[nextWalletID] = wallet
	nextWalletID++
	http.Redirect(w, r, "/famoney/dashboard", http.StatusSeeOther)
}

func viewWalletHandler(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/famoney/wallet/")
	id, _ := strconv.Atoi(idStr)
	wallet, ok := wallets[id]
	if !ok {
		http.NotFound(w, r)
		return
	}
	if r.Method == "POST" {
		action := r.FormValue("action")
		amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
		categoryID, _ := strconv.Atoi(r.FormValue("category"))
		desc := r.FormValue("description")
		if action == "flow" {
			wallet.Balance += amount
			wallet.CategoryBalances[categoryID] += amount
			flows[nextFlowID] = &Flow{ID: nextFlowID, WalletID: wallet.ID, Amount: amount, Currency: wallet.Currency, CategoryID: categoryID, Description: desc, CreatedAt: time.Now()}
			nextFlowID++
		} else if action == "balance" {
			diff := amount - wallet.Balance
			wallet.Balance = amount
			wallet.CategoryBalances[categoryID] += diff
			flows[nextFlowID] = &Flow{ID: nextFlowID, WalletID: wallet.ID, Amount: diff, Currency: wallet.Currency, CategoryID: categoryID, Description: desc, CreatedAt: time.Now()}
			nextFlowID++
		}
	}
	walletFlows := []*Flow{}
	for _, f := range flows {
		if f.WalletID == wallet.ID {
			walletFlows = append(walletFlows, f)
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
	categories[nextCategoryID] = &Category{ID: nextCategoryID, Name: name}
	nextCategoryID++
	http.Redirect(w, r, "/famoney/dashboard", http.StatusSeeOther)
}
