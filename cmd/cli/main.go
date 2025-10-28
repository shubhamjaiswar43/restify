package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"syscall"
	"text/tabwriter"

	"golang.org/x/term"
)

var baseURL = "http://localhost:8082"
var dineInUserID = "690069c40c0d686e011c80b8"
var jwtToken string
var userRole string

type Restaurant struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
}

type MenuItem struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type Order struct {
	ID           string      `json:"id"`
	UserID       string      `json:"user_id"`
	RestaurantID string      `json:"restaurant_id"`
	Status       string      `json:"status"`
	TotalPrice   float64     `json:"total_price"`
	Items        []OrderItem `json:"items"`
}

type OrderItem struct {
	MenuItemID string  `json:"menu_item_id"`
	Quantity   int     `json:"quantity"`
	Price      float64 `json:"price"`
}

func main() {
	for {
		fmt.Println("\n--- Restaurant Management CLI ---")
		fmt.Println("1. Signup")
		fmt.Println("2. Login")
		fmt.Println("3. Exit")
		fmt.Print("Choose option: ")

		var choice int
		fmt.Scanln(&choice)

		switch choice {
		case 1:
			signup()
		case 2:
			if login() {
				adminPanel()
			}
		case 3:
			fmt.Println("Goodbye!")
			return
		default:
			fmt.Println("Invalid choice, try again.")
		}
	}
}

func signup() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter name: ")
	name, _ := reader.ReadString('\n')
	fmt.Print("Enter email: ")
	email, _ := reader.ReadString('\n')
	fmt.Print("Enter password: ")
	passwordBytes, _ := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	password := string(passwordBytes)
	fmt.Print("Enter role (admin/customer): ")
	role, _ := reader.ReadString('\n')

	payload := map[string]string{
		"name":     strings.TrimSpace(name),
		"email":    strings.TrimSpace(email),
		"password": strings.TrimSpace(password),
		"role":     strings.TrimSpace(role),
	}
	data, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", baseURL+"/signup", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	if role == "admin" {
		fmt.Print("Enter Admin-Secret: ")
		adminKey, _ := reader.ReadString('\n')
		req.Header.Set("Admin-Secret", strings.TrimSpace(adminKey))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Request failed:", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
}

func login() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter email: ")
	email, _ := reader.ReadString('\n')
	fmt.Print("Enter password: ")
	passwordBytes, _ := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	password := string(passwordBytes)

	payload := map[string]string{
		"email":    strings.TrimSpace(email),
		"password": strings.TrimSpace(password),
	}
	data, _ := json.Marshal(payload)

	resp, err := http.Post(baseURL+"/login", "application/json", bytes.NewBuffer(data))
	if err != nil {
		fmt.Println("Request failed:", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("Login failed:", string(body))
		return false
	}

	var res map[string]string
	json.NewDecoder(resp.Body).Decode(&res)
	jwtToken = res["token"]
	userRole = res["role"]

	fmt.Println("\nLogin successful! Role:", userRole)
	return true
}

func adminPanel() {
	for {
		fmt.Println("\n--- Admin Panel ---")
		fmt.Println("1. List Restaurants")
		fmt.Println("2. Add Restaurant")
		fmt.Println("3. Select Restaurant")
		fmt.Println("4. Logout")
		fmt.Print("Choose option: ")

		var choice int
		fmt.Scanln(&choice)

		switch choice {
		case 1:
			listRestaurants()
		case 2:
			addRestaurant()
		case 3:
			selectRestaurant()
		case 4:
			fmt.Println("Logged out.")
			jwtToken, userRole = "", ""
			return
		default:
			fmt.Println("Invalid choice.")
		}
	}
}

func listRestaurants() []Restaurant {
	req, _ := http.NewRequest("GET", baseURL+"/restaurants", nil)
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Request failed:", err)
		return nil
	}
	defer resp.Body.Close()

	var parsed struct {
		Restaurants []Restaurant `json:"restaurants"`
	}
	json.NewDecoder(resp.Body).Decode(&parsed)

	if len(parsed.Restaurants) == 0 {
		fmt.Println("No restaurants found.")
		return nil
	}

	fmt.Println("\nRestaurants:")
	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, "No\tID\tName\tAddress\tPhone")
	for i, r := range parsed.Restaurants {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", i+1, r.ID, r.Name, r.Address, r.Phone)
	}
	w.Flush()
	return parsed.Restaurants
}

func addRestaurant() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter restaurant name: ")
	name, _ := reader.ReadString('\n')
	fmt.Print("Enter address: ")
	address, _ := reader.ReadString('\n')
	fmt.Print("Enter phone: ")
	phone, _ := reader.ReadString('\n')

	payload := map[string]string{
		"name":    strings.TrimSpace(name),
		"address": strings.TrimSpace(address),
		"phone":   strings.TrimSpace(phone),
	}
	data, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", baseURL+"/restaurants", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwtToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Request failed:", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
}

func selectRestaurant() {
	restaurants := listRestaurants()
	if len(restaurants) == 0 {
		return
	}

	fmt.Print("\nEnter restaurant number: ")
	var idx int
	fmt.Scanln(&idx)
	if idx < 1 || idx > len(restaurants) {
		fmt.Println("Invalid choice.")
		return
	}

	selected := restaurants[idx-1]
	restaurantPanel(selected)
}

func restaurantPanel(r Restaurant) {
	for {
		fmt.Printf("\n--- Restaurant: %s ---\n", r.Name)
		fmt.Println("1. List Menu")
		fmt.Println("2. Add Menu")
		fmt.Println("3. List Orders")
		fmt.Println("4. Add Order")
		fmt.Println("5. Back")
		fmt.Print("Choose option: ")

		var choice int
		fmt.Scanln(&choice)

		switch choice {
		case 1:
			listMenu(r.ID)
		case 2:
			addMenu(r.ID)
		case 3:
			listOrders(r.ID)
		case 4:
			addOrder(r.ID)
		case 5:
			return
		default:
			fmt.Println("Invalid choice.")
		}
	}
}

func listMenu(restaurantID string) []MenuItem {
	url := fmt.Sprintf("%s/menu-items?restaurant_id=%s", baseURL, restaurantID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Request failed:", err)
		return nil
	}
	defer resp.Body.Close()

	var parsed struct {
		MenuItems []MenuItem `json:"menu_items"`
	}
	json.NewDecoder(resp.Body).Decode(&parsed)

	if len(parsed.MenuItems) == 0 {
		fmt.Println("No menu items found.")
		return nil
	}

	fmt.Println("\nMenu Items:")
	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, "No\tID\tName\tCategory\tPrice\tDescription")
	for i, m := range parsed.MenuItems {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%.2f\t%s\n", i+1, m.ID, m.Name, m.Category, m.Price, m.Description)
	}
	w.Flush()
	return parsed.MenuItems
}

func addMenu(restaurantID string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter menu name: ")
	name, _ := reader.ReadString('\n')
	fmt.Print("Enter category: ")
	category, _ := reader.ReadString('\n')
	fmt.Print("Enter description: ")
	description, _ := reader.ReadString('\n')
	fmt.Print("Enter price: ")
	var price float64
	fmt.Scanln(&price)

	payload := map[string]interface{}{
		"name":          strings.TrimSpace(name),
		"category":      strings.TrimSpace(category),
		"description":   strings.TrimSpace(description),
		"price":         price,
		"restaurant_id": restaurantID,
	}
	data, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", baseURL+"/menu-items", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwtToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Request failed:", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
}

func listOrders(restaurantID string) {
	url := fmt.Sprintf("%s/orders/?restaurant_id=%s", baseURL, restaurantID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Request failed:", err)
		return
	}
	defer resp.Body.Close()

	var parsed struct {
		Orders []Order `json:"orders"`
	}
	json.NewDecoder(resp.Body).Decode(&parsed)

	if len(parsed.Orders) == 0 {
		fmt.Println("No orders found.")
		return
	}

	fmt.Println("\nOrders:")
	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, "No\tID\tStatus\tTotal Price\tItems")
	for i, o := range parsed.Orders {
		fmt.Fprintf(w, "%d\t%s\t%s\t%.2f\t%d items\n", i+1, o.ID, o.Status, o.TotalPrice, len(o.Items))
	}
	w.Flush()
}

func addOrder(restaurantID string) {
	menuItems := listMenu(restaurantID)
	if len(menuItems) == 0 {
		fmt.Println("No menu available to order.")
		return
	}

	fmt.Print("\nEnter menu number: ")
	var idx int
	fmt.Scanln(&idx)
	if idx < 1 || idx > len(menuItems) {
		fmt.Println("Invalid choice.")
		return
	}
	selected := menuItems[idx-1]

	fmt.Print("Enter quantity: ")
	var qty int
	fmt.Scanln(&qty)

	payload := map[string]interface{}{
		"user_id":       dineInUserID,
		"restaurant_id": restaurantID,
		"items": []map[string]interface{}{
			{
				"menu_item_id": selected.ID,
				"quantity":     qty,
				"price":        selected.Price,
			},
		},
		"total_price": selected.Price * float64(qty),
		"status":      "preparing",
	}
	data, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", baseURL+"/orders", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwtToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Request failed:", err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
}
