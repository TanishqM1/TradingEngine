#include "httplib.h"
#include <iostream>
#include <string>
#include <map>
#include <unordered_map>
#include <set>
#include <list>
#include <cmath>
#include <ctime>
#include <cstdint>
#include <vector>
#include <format>
#include <stdexcept>
#include <memory>
#include <iterator>
#include <numeric>
#include <atomic>
#include <mutex>

using namespace std;

// "Order"s will have two Time Enforcement options.
enum class OrderType{
    GoodTillCancel,
    FillAndKill
};

// "Order"s will have a Side. Side::Buy or Side::Sell
enum class Side{
    Buy,
    Sell
};

// alias types
using Price = std::int32_t; // price can be negative
using Quantity = std::uint32_t;
using OrderId = std::uint64_t;

// in cpp we denote "member varaibles" (i.e not parameters) with a "_".

struct LevelInfo{
    Price price_;
    Quantity quantity_;
};

// LevelInfos stores all the Quantity's at a certain price (level).
using LevelInfos = std::vector<LevelInfo>;

// OrderBookLevelInfo stores the vectors for asks and bids for all prices.
class OrderBookLevelInfo{
    // we need seperate vectors for bids and asks
    public:
        OrderBookLevelInfo(const LevelInfos& asks, const LevelInfos& bids):
        // constructor instantiation
        asks_(asks),
        bids_(bids) {}

        const LevelInfos& GetBids() const { return bids_; }
        const LevelInfos& GetAsks() const { return asks_; }
    
    private:
        LevelInfos bids_;
        LevelInfos asks_;
};

// What is added to the order book? Objets that have the order type, key, side, price, quantity, and bool(s) for filled or not
// Order stores instances of an order (with all needed properties).
class Order {
    // A PUBLIC constructor can initialize private fields.
    public:
        Order(OrderType orderType, Side side, Price price, Quantity quantity, OrderId orderId): 
            orderType_(orderType),
            orderId_(orderId),
            price_(price),
            side_(side),
            initialQuantity_(quantity),
            remainingQuantity_(quantity) {}

        // const in the function sig. means it will NOT alter the members (getts and setters, bools).
        OrderId GetOrderId() const { return orderId_; }
        Side GetSide() const { return side_; }
        Price GetPrice() const { return price_; }
        OrderType GetOrderType() const { return orderType_; }
        Quantity GetInitialQuantity() const { return initialQuantity_; }
        Quantity GetRemainingQuantity() const { return remainingQuantity_; }
        Quantity FilledQuantity() const { return GetInitialQuantity() - GetRemainingQuantity();}
        bool IsFilled() const { return GetRemainingQuantity() == 0;}

        void Fill(Quantity quantity){
            // validate if the # of orders can actually be filled
            if (quantity > GetRemainingQuantity()){
                throw std::logic_error(std::format("Order ({}) cannot be filled for more than it's remaining quantity", GetOrderId()));
            }

            remainingQuantity_ -= quantity; // it has been filled
        }


        // the reason we need this private section here is because without it, we declare the variables in our public: modifier, but never assign them a type.
    private:
        OrderType orderType_;
        OrderId orderId_;
        Price price_;
        Side side_;
        Quantity initialQuantity_;
        Quantity remainingQuantity_;
};

// Orders go into multiple data structures, so we will keep a pointer to orders. (reference semantics) so we can easily reference them.
// std::make_shared<type>(); allocates order(s) on the heap, and returns a pointer pointing at it.

// So here, we have a vector of POINTERS to orders.
using OrderPointer = std::shared_ptr<Order>;
using OrderPointers = std::list<OrderPointer>; // we use a LIST because if we have orders at the same price, we want a FIFO order.

// Common functionality we need to support for orders:

// Add() => we need a new order.
// Cancel() => we need an existing valid order id.
// Modify() => we need a way to modify existing orders, and we need to retrieve orders in a well manner.

class OrderModify{
    public:
        OrderModify(OrderId orderId, Side side, Price price, Quantity quantity):
        orderId_(orderId),
        side_(side),
        price_(price),
        quantity_(quantity) {}

    OrderId GetOrderId() const {return orderId_;}
    Price GetPrice() const {return price_;}
    Side GetSide() const {return side_;}
    Quantity GetQuantity() const {return quantity_;}

    // "const" in this function denotes the function does NOT modify any member variables.
    OrderPointer ToOrderPointer(OrderType type) const {
    return std::make_shared<Order>(type, GetSide(), GetPrice(), GetQuantity(), GetOrderId());
}

    private:
        OrderId orderId_;
        Side side_;
        Price price_;
        Quantity quantity_;
};

// TradeInfo may exist by itself, but Trade can contain or reference one or more TradeInfo objects.
struct TradeInfo{
    OrderId orderid_;
    Price price_;
    Quantity quantity_;
};

// A trade consists of a bid and ask, which will hava TradeInfo objects for eahch.
class Trade{
    public:

        Trade(const TradeInfo& bidTrade, const TradeInfo& askTrade):
        bidTrade_ { bidTrade},
        askTrade_ { askTrade} {}

        const TradeInfo& GetBidTrade() const {return bidTrade_;}
        const TradeInfo& GetAskTrade() const {return askTrade_;}

    private:
        TradeInfo bidTrade_;
        TradeInfo askTrade_;
};


// vector of trade object, representing bids and asks
using Trades = std::vector<Trade>;

class Orderbook{
    // An OrderBook holds orders, and we want to be easily able to access these orders (preferrable, in O(1) time). Any any point in time, the bids and asks we are about are:
    // The bid with the HIGHEST price, and the ask with the LOWEST price.

    private:
        // when an entry is to be ordered, we take the pointer to the specified entries.
        struct OrderEntry{
            OrderPointer order_ { nullptr };
            OrderPointers::iterator location_;
        };

        // hashmap of key Price, and mapped value 'OrderPointers'. std::greater<Price> is a custom comparator to sort upon, where it's in descending order. (highest ASK first!).
        std::map<Price, OrderPointers, std::greater<Price>> bids_;
        std::map<Price, OrderPointers, std::less<Price>> asks_;
        // we don't need to sort our actual orders. these are just for the record.
        std::unordered_map<OrderId, OrderEntry> orders_;

        // We need CanMatch() for fillandkill orders, because if it's can't match now, we never do it (now or never).
        // otherwise, if we have a goodtillcancel order, we can add it to the orderbook, and then match it when possible.
        // Upon match, we need to REMOVE the order from the orderbook. This may be completely remaining orders, or partially filled orders.

        bool CanMatch(Side side, Price price) const{
              if (side == Side::Buy){

                if (asks_.empty()){
                    return false;
                }else{
                    const auto& [bestAsk, _] = *asks_.begin(); // starts at the best ask (lowest price!).
                    return price >= bestAsk; // we return the best match possible, and return if it is valid or not.
                }
              }

            //   copying for other side
              if (side == Side::Sell){
                if (bids_.empty()){
                    return false;
                }else{
                    const auto& [bestBid, _] = *bids_.begin();
                    return price <= bestBid;
                }
              }

              return false; // Default case (should never reach here)
          }

        // We also need a Match() function that runs when a match actually occurs. 
        // Replace your MatchOrders() function with this debug version:

Trades MatchOrders(){
    std::cout << "\n[DEBUG] MatchOrders called" << std::flush;
    Trades trades;
    trades.reserve(orders_.size());

    std::cout << "\n[DEBUG] Bids empty: " << bids_.empty() << ", Asks empty: " << asks_.empty() << std::flush;

    while (true){
        if (bids_.empty() || asks_.empty()){ 
            std::cout << "\n[DEBUG] One side empty, breaking" << std::flush;
            break;
        }
        
        std::cout << "\n[DEBUG] Getting best bid and ask" << std::flush;
        auto& [askPrice, asks] = *asks_.begin();
        auto& [bidPrice, bids] = *bids_.begin();
        
        std::cout << "\n[DEBUG] BidPrice: " << bidPrice << ", AskPrice: " << askPrice << std::flush;
    
        if (bidPrice < askPrice){ 
            std::cout << "\n[DEBUG] No match possible, breaking" << std::flush;
            break; 
        }

        std::cout << "\n[DEBUG] Starting match loop" << std::flush;
        while (!bids.empty() && !asks.empty()){
            std::cout << "\n[DEBUG] Getting front orders" << std::flush;
            auto& bid = bids.front();
            auto& ask = asks.front();

            std::cout << "\n[DEBUG] Bid ID: " << bid->GetOrderId() << ", Ask ID: " << ask->GetOrderId() << std::flush;

            Quantity quantity = std::min(bid->GetRemainingQuantity(), ask->GetRemainingQuantity());
            std::cout << "\n[DEBUG] Matching quantity: " << quantity << std::flush;
            
            bid->Fill(quantity);
            ask->Fill(quantity);
            
            std::cout << "\n[DEBUG] Creating trade log" << std::flush;
            trades.push_back(Trade{
                TradeInfo{ bid->GetOrderId(), bid->GetPrice(), quantity},
                TradeInfo{ ask->GetOrderId(), ask->GetPrice(), quantity}
            });
            
            std::cout << "\n[DEBUG] Checking if orders filled" << std::flush;
            if (bid->IsFilled()){
                std::cout << "\n[DEBUG] Removing filled bid" << std::flush;
                OrderId bidId = bid->GetOrderId();
                bids.pop_front();
                orders_.erase(bidId);
            }
            if (ask->IsFilled()){
                std::cout << "\n[DEBUG] Removing filled ask" << std::flush;
                OrderId askId = ask->GetOrderId();
                asks.pop_front();
                orders_.erase(askId);
            }
        }
        
        std::cout << "\n[DEBUG] Inner loop done, checking empty" << std::flush;
        if (bids.empty()){ 
            std::cout << "\n[DEBUG] Erasing bid price level" << std::flush;
            bids_.erase(bidPrice);
        }
        if (asks.empty()){ 
            std::cout << "\n[DEBUG] Erasing ask price level" << std::flush;
            asks_.erase(askPrice);
        }
    }

    std::cout << "\n[DEBUG] Main matching done, checking FillAndKill" << std::flush;

    // Handle FillAndKill orders that didn't fully fill
    if (!bids_.empty()){
        std::cout << "\n[DEBUG] Checking bids for FillAndKill" << std::flush;
        auto bidIter = bids_.begin();
        auto& [_, bidsRef] = *bidIter;
        if (!bidsRef.empty()) {
            auto& order = bidsRef.front();
            if (order->GetOrderType() == OrderType::FillAndKill && !order->IsFilled()){
                std::cout << "\n[DEBUG] Canceling unfilled FillAndKill bid" << std::flush;
                OrderId orderId = order->GetOrderId();
                CancelOrder(orderId);
            }
        }
    }

    if (!asks_.empty()){
        std::cout << "\n[DEBUG] Checking asks for FillAndKill" << std::flush;
        auto askIter = asks_.begin();
        auto& [_, asksRef] = *askIter;
        if (!asksRef.empty()) {
            auto& order = asksRef.front();
            if (order->GetOrderType() == OrderType::FillAndKill && !order->IsFilled()){
                std::cout << "\n[DEBUG] Canceling unfilled FillAndKill ask" << std::flush;
                OrderId orderId = order->GetOrderId();
                CancelOrder(orderId);
            }
        }
    }
    
    std::cout << "\n[DEBUG] MatchOrders returning" << std::flush;
    return trades;
}

        // need to add, cancel, and modify order(s).

        // Given a new Order (the pointer to it), this method adds it to our orderbook.
        // it checks if the order already exists, if the order is a fillandkill and can NOT be immediately matched (both cases where we do NOT add).
        public:
            Trades AddOrder(OrderPointer order){
                if (orders_.contains(order->GetOrderId())){ return { };}

                if (order->GetOrderType() == OrderType::FillAndKill && !CanMatch(order->GetSide(), order->GetPrice())){
                    return { };
                }
                
                // iteator to OrderPointers, which is simply a LIST. allows access for O(1) remove/cancellation.
                // bids_ is our buy-side storage, whereas asks_ is our sell-side storage.
                OrderPointers::iterator iterator;

                if (order->GetSide() == Side::Buy){
                    auto& orders = bids_[order->GetPrice()]; 
                    // this line causes INSERTION, where the price of the order is used as the key, and simultaneously gives an "orders" alias which is the list of orders at the specific price level.
                    // so we insert an order (with Price as the key) and retrieve the reference to the list (value).
                    orders.push_back(order);
                    iterator = std::next(orders.begin(), orders.size()-1);
                    // the order is added to the back of the list (FIFO), and the iterator wil calculate the exact position of the order we just inserted. (for O(1) removal later if needed).
                }else{
                    auto& orders = asks_[order->GetPrice()];
                    orders.push_back(order);
                    iterator = std::next(orders.begin(), orders.size()-1);
                }

                // general bookkeeping in the orders_ OrderBook.
                orders_.insert({order->GetOrderId(), OrderEntry{ order, iterator}});
                return MatchOrders();
            }
            
            // method to REMOVE an order from the orderbook if it is cancelled.
            void CancelOrder(OrderId orderId){
            if (!orders_.contains(orderId)){
                return;
            }
            // we need aliases to the order and iterator (retrieved from orders_ using the orderId). Then we can remove it from the orders_.
            const auto& [order, orderIterator] = orders_.at(orderId);
            orders_.erase(orderId);

            // if it's a sell order, we remove it from the asks_ data structure. if it's empty after, we need to remove the price altogether from it (memory cleanup).

            if (order->GetSide() == Side::Sell){
                auto price = order->GetPrice();
                auto& orders = asks_.at(price);
                orders.erase(orderIterator);
                if (orders.empty()){
                    asks_.erase(price);
                }
            }else{
                auto price = order->GetPrice();
                auto& orders = bids_.at(price);
                orders.erase(orderIterator);
                if (orders.empty()){
                    bids_.erase(price);
                }
            }}

            
            Trades MatchOrder(OrderModify order){
                if (!orders_.contains(order.GetOrderId())){
                    return { };
                }

                // fetch information of an order, cancel the order, and add the modified version back.
                const auto& [existingOrder, _] = orders_.at(order.GetOrderId());
                CancelOrder(order.GetOrderId());
                return AddOrder(order.ToOrderPointer(existingOrder->GetOrderType()));
            }

            std::size_t Size() const { return orders_.size();}

            // Clear all orders from the orderbook
            void Clear() {
                bids_.clear();
                asks_.clear();
                orders_.clear();
            }

            // Get the best bid and ask prices (-1 if empty)
            std::pair<Price, Price> GetBestPrices() const {
                Price bestBid = bids_.empty() ? -1 : bids_.begin()->first;
                Price bestAsk = asks_.empty() ? -1 : asks_.begin()->first;
                return {bestBid, bestAsk};
            }

            // Count remaining bids and asks
            std::pair<std::size_t, std::size_t> GetOrderCounts() const {
                std::size_t bidCount = 0;
                std::size_t askCount = 0;
                for (const auto& [price, orders] : bids_) {
                    bidCount += orders.size();
                }
                for (const auto& [price, orders] : asks_) {
                    askCount += orders.size();
                }
                return {bidCount, askCount};
            }

            // Get number of price levels
            std::pair<std::size_t, std::size_t> GetLevelCounts() const {
                return {bids_.size(), asks_.size()};
            }

            OrderBookLevelInfo GetOrderInfos() const{
                // alias for a LevelInfo vector, and we allocate memory in each LevelInfos (orders_ is conservative, we can use asks_ and bids_ if we really wanted to).
                LevelInfos askinfos, bidinfos;
                bidinfos.reserve(orders_.size());
                askinfos.reserve(orders_.size());

                // this is a lambda function that takes a Price and list of OrderPointers at that price, and returns a LevelInfo struct containing all of them (struct has Price and TotalQuantity).
                // accumulate iterates from orders.start to orders.end, starts with a value of 0, and adds the sum of OrderPointer() in an order.'

                // so within OrderPointers -> OrderPointer -> OrderPointer Quantity is what we want the sum of. Tells us how many shares are "up for consideration".
                auto CreateLevelInfos = [](Price price, const OrderPointers& orders){
                    return LevelInfo{ price, std::accumulate(orders.begin(), orders.end(), (Quantity)0, [](std::size_t runningSum, const OrderPointer& order){
                        return runningSum + order->GetRemainingQuantity();
                    })};
                };

                // finally, for each pricelevel in bids_, we take the pricelevel & OrderPointers (which point to all the live orders)
                // we calcualte the total sum/quantity of shares in all orders at the price level COMBINED.
                // push that number back to bidinfos and askinfos.
                for (const auto& [price, orders] : bids_)
                    bidinfos.push_back(CreateLevelInfos(price, orders));
                
                for (const auto& [price, orders] : asks_)
                    askinfos.push_back(CreateLevelInfos(price, orders));
                // in the end, bidinfos and askinfos is a vector of the "LevelInfo" object, which stores price-totalquantity pair(s). 
                // helps us find the liquidity of shares at certain prices, using asks/bids.
                return OrderBookLevelInfo(askinfos, bidinfos);
            }

};

OrderType setType(string type){
    if (type == "goodtillcancel"){
        return OrderType::GoodTillCancel;
    }else{
        return OrderType::FillAndKill;
    }
}

Side setSide(string side){
    if (side == "buy"){
        return Side::Buy;
    }else{
        return Side::Sell;
    }
}

struct Counter{
    uint64_t count = 0;

    uint64_t GetNext(){
        return ++count;
    }
    uint64_t GetCurrent(){
        return count;
    }
};

std::mutex gLock;
std::unordered_map<string, Orderbook> MyMap;

OrderType parse_ordertype(string type){
    if (type == "GTC"){return OrderType::GoodTillCancel;}
    else{return OrderType::FillAndKill;}
}

Side parse_side(string side){
    if (side=="BUY"){return Side::Buy;}
    else{return Side::Sell;}
}

// Use stoull (string to unsigned long long) for OrderId (uint64_t)
OrderId parse_id(string id){
    // This supports values up to 18 quintillion (uint64_t max)
    uint64_t res = std::stoull(id); 
    return res;
}

// Use stoul (string to unsigned long) for Quantity (uint32_t)
Quantity parse_quantity(string quantity){
    // This supports values up to 4.2 billion (uint32_t max)
    uint32_t res = static_cast<uint32_t>(std::stoul(quantity));
    return res;
}

// Keep parse_price as stoi since Price (int32_t) is a signed 32-bit integer.
Price parse_price(string price){
    return std::stoi(price);
}

// JSON parsing helpers for batch endpoint
std::string extract_json_string(const std::string& json, const std::string& key) {
    std::string search_key = "\"" + key + "\":";
    size_t key_pos = json.find(search_key);
    if (key_pos == std::string::npos) return "";

    size_t start = json.find("\"", key_pos + search_key.length());
    if (start == std::string::npos) return "";
    start++;

    size_t end = json.find("\"", start);
    if (end == std::string::npos) return "";

    return json.substr(start, end - start);
}

int64_t extract_json_number(const std::string& json, const std::string& key) {
    std::string search_key = "\"" + key + "\":";
    size_t key_pos = json.find(search_key);
    if (key_pos == std::string::npos) return 0;

    size_t start = key_pos + search_key.length();
    // Skip whitespace
    while (start < json.length() && (json[start] == ' ' || json[start] == '\t')) start++;

    size_t end = start;
    while (end < json.length() && (isdigit(json[end]) || json[end] == '-')) end++;

    if (start == end) return 0;
    return std::stoll(json.substr(start, end - start));
}

std::vector<std::string> extract_json_array(const std::string& json, const std::string& key) {
    std::vector<std::string> result;
    std::string search_key = "\"" + key + "\":";
    size_t key_pos = json.find(search_key);
    if (key_pos == std::string::npos) return result;

    size_t array_start = json.find("[", key_pos);
    if (array_start == std::string::npos) return result;

    size_t array_end = json.find("]", array_start);
    if (array_end == std::string::npos) return result;

    // Parse individual objects in the array
    size_t pos = array_start + 1;
    while (pos < array_end) {
        size_t obj_start = json.find("{", pos);
        if (obj_start == std::string::npos || obj_start >= array_end) break;

        // Find matching closing brace
        int brace_count = 1;
        size_t obj_end = obj_start + 1;
        while (obj_end < json.length() && brace_count > 0) {
            if (json[obj_end] == '{') brace_count++;
            else if (json[obj_end] == '}') brace_count--;
            obj_end++;
        }

        if (brace_count == 0) {
            result.push_back(json.substr(obj_start, obj_end - obj_start));
        }
        pos = obj_end;
    }

    return result;
}

// Structure to track batch statistics per book
struct BookStats {
    int tradesExecuted = 0;
    int64_t volumeTraded = 0;
};


void server_trade(const httplib::Request& req, httplib::Response& res){
    try{
        // parse content'
        string s_orderid = req.get_param_value("orderid");
        string s_type = req.get_param_value("tradetype");
        string s_side = req.get_param_value("side");
        string s_price = req.get_param_value("price");
        string s_quantity = req.get_param_value("quantity");
        string s_book = req.get_param_value("book");

        if (s_book.empty() || s_orderid.empty() || s_type.empty() || s_side.empty() || s_price.empty() || s_quantity.empty()) {
                    res.status = 400; // Bad Request
                    res.set_content(R"({"error":"Missing required parameters"})", "application/json");
                    return;
                }
        OrderId id = parse_id(s_orderid);
        OrderType type = parse_ordertype(s_type);
        Side side = parse_side(s_side);
        Price price = parse_price(s_price);
        Quantity quantity = parse_quantity(s_quantity);

        {
        std::lock_guard<std::mutex> lock(gLock);;
        Orderbook& book = MyMap[s_book];
        book.AddOrder(std::make_shared<Order>(type, side, price, quantity, id));
        
        cout << "\n " << book.Size();
        }
        res.status = 200; // or httplib::StatusCode::OK_200
        res.set_content("{\"message\": \"Order placed successfully\"}", "application/json");
    }catch(const std::exception& e) {
        // Catch standard C++ errors (like bad numeric conversion)
        res.status = 500; // Internal Server Error is better for conversion errors
        std::cerr << "Error in server_trade: " << e.what() << std::endl;
        res.set_content(std::format(R"({{"error":"Engine error during processing: {}"}})", e.what()), "application/json");
    } catch(...) {
        // Catch-all for unknown errors
        res.status = 500; 
        res.set_content(R"({"error":"Unknown internal server error."})", "application/json");
    }

}

// Note: This assumes the global map std::unordered_map<string, Orderbook> MyMap;
// and conversion functions like parse_id are globally defined.

void server_cancel(const httplib::Request& req, httplib::Response& res) {
    try{
        // parse content
        string s_orderid = req.get_param_value("orderid");
        string s_book = req.get_param_value("book");

        if (s_orderid.empty() || s_book.empty()){
            res.status = 400; // Bad Request
            res.set_content(R"({"error":"Missing required parameters"})", "application/json");
            return;
        }

        OrderId id = parse_id(s_orderid);
        
        std::lock_guard<std::mutex> lock(gLock);
        Orderbook& book = MyMap[s_book];
        size_t before = book.Size();
        book.CancelOrder(id);
        size_t after = book.Size();
        

        if (after < before){
            cout << "\n " << s_orderid << " " << s_book;
        res.status = 200;
        res.set_content("{\"message\": \"Order Info Received\"}", "application/json");
        cout << "\n Cancelled OrderID: " << s_orderid << " in book: " << s_book << " new size:  " << book.Size();
        }else {
            res.status = 404;
            res.set_content("{\"message\": \"Order ID not found\"}", "application/json");
        }
    }catch(...){
        res.status = 500;
        res.set_content(R"({"error":"Unknown internal server error."})", "application/json");
        cout << "\n There was an error";
    }
}

std::string level_infos_to_json(const OrderBookLevelInfo& info, size_t size) {
    auto convert_levels = [](const LevelInfos& levels, const std::string& type) {
        std::string json_array = "[";
        bool first = true;
        for (const auto& level : levels) {
            if (!first) {
                json_array += ",";
            }
            json_array += std::format(
                R"({{"type":"{}", "price":{}, "quantity":{}}})",
                type, level.price_, level.quantity_
            );
            first = false;
        }
        json_array += "]";
        return json_array;
    };

    return std::format(
        R"({{"bids":{}, "asks":{}, "size":{}}})",
        convert_levels(info.GetBids(), "Bid"),
        convert_levels(info.GetAsks(), "Ask"),
        size
    );
}

// NOTE: This relies on the OrderBookLevelInfo, LevelInfos, Price, and Quantity types being correctly defined earlier in Server.cpp.

std::string all_orderbooks_to_json() {
    std::string json_output = "{";
    bool first = true;
    
    // MyMap is the global std::unordered_map<string, Orderbook>
    for (const auto& pair : MyMap) { 
        if (!first) {
            json_output += ",";
        }
        std::string book_name = pair.first;
        const Orderbook& book = pair.second;

        // Uses the existing utility to get the JSON for one book
        std::string book_json_content = level_infos_to_json(book.GetOrderInfos(), book.Size());
        
        // Format the book name as the key, and insert the book's JSON content
        // We remove the outer braces from book_json_content to embed it correctly
        json_output += std::format(R"("{}":{})", book_name, book_json_content);
        first = false;
    }
    json_output += "}";
    return json_output;
}

void server_status(const httplib::Request& req, httplib::Response& res) {
    try {
        std::lock_guard<std::mutex> lock(gLock);
        std::string status_json = all_orderbooks_to_json();

        res.set_content(status_json, "application/json");
        res.status = 200;
    } catch (const std::exception& e) {
        res.status = 500;
        std::cerr << "Exception in server_status: " << e.what() << std::endl;
        res.set_content(std::format(R"({{"error":"Engine error getting status: {}"}})", e.what()), "application/json");
    } catch (...) {
        res.status = 500;
        res.set_content(R"({"error":"Unknown internal server error getting status."})", "application/json");
    }
}

void server_reset(const httplib::Request& req, httplib::Response& res) {
    try {
        std::lock_guard<std::mutex> lock(gLock);
        size_t count = MyMap.size();
        MyMap.clear();

        res.status = 200;
        res.set_content(std::format(R"({{"message":"All orderbooks cleared","booksCleared":{}}})", count), "application/json");
        std::cout << "\n[RESET] Cleared " << count << " orderbooks" << std::flush;
    } catch (const std::exception& e) {
        res.status = 500;
        std::cerr << "Exception in server_reset: " << e.what() << std::endl;
        res.set_content(std::format(R"({{"error":"Engine error during reset: {}"}})", e.what()), "application/json");
    } catch (...) {
        res.status = 500;
        res.set_content(R"({"error":"Unknown internal server error during reset."})", "application/json");
    }
}

void server_batch(const httplib::Request& req, httplib::Response& res) {
    try {
        // Parse the JSON body
        std::string body = req.body;
        std::vector<std::string> orders = extract_json_array(body, "orders");

        if (orders.empty()) {
            res.status = 400;
            res.set_content(R"({"error":"No orders provided in batch"})", "application/json");
            return;
        }

        // Track statistics per book
        std::unordered_map<std::string, BookStats> bookStats;
        int processedCount = 0;

        {
            std::lock_guard<std::mutex> lock(gLock);

            // Process each order
            for (const auto& orderJson : orders) {
                OrderId id = static_cast<OrderId>(extract_json_number(orderJson, "orderid"));
                std::string book = extract_json_string(orderJson, "book");
                std::string typeStr = extract_json_string(orderJson, "tradetype");
                std::string sideStr = extract_json_string(orderJson, "side");
                Price price = static_cast<Price>(extract_json_number(orderJson, "price"));
                Quantity quantity = static_cast<Quantity>(extract_json_number(orderJson, "quantity"));

                if (book.empty() || id == 0) continue;

                OrderType type = parse_ordertype(typeStr);
                Side side = parse_side(sideStr);

                Orderbook& orderbook = MyMap[book];
                Trades trades = orderbook.AddOrder(std::make_shared<Order>(type, side, price, quantity, id));

                // Track statistics
                BookStats& stats = bookStats[book];
                stats.tradesExecuted += trades.size();
                for (const auto& trade : trades) {
                    stats.volumeTraded += trade.GetBidTrade().quantity_;
                }

                processedCount++;
            }

            // Build response JSON with per-book results
            std::string resultJson = "{";
            resultJson += std::format(R"("processedCount":{},"results":{{)", processedCount);

            bool first = true;
            for (const auto& [bookName, book] : MyMap) {
                if (!first) resultJson += ",";
                first = false;

                auto [bestBid, bestAsk] = book.GetBestPrices();
                auto [bidCount, askCount] = book.GetOrderCounts();
                auto [bidLevels, askLevels] = book.GetLevelCounts();

                // Get stats for this book (may be zero if no orders were for this book)
                const BookStats& stats = bookStats[bookName];

                resultJson += std::format(
                    R"("{}":{{"tradesExecuted":{},"volumeTraded":{},"remainingBids":{},"remainingAsks":{},"bestBidPrice":{},"bestAskPrice":{},"bidLevels":{},"askLevels":{}}})",
                    bookName,
                    stats.tradesExecuted,
                    stats.volumeTraded,
                    bidCount,
                    askCount,
                    bestBid,
                    bestAsk,
                    bidLevels,
                    askLevels
                );
            }

            resultJson += "}}";

            res.status = 200;
            res.set_content(resultJson, "application/json");
            std::cout << "\n[BATCH] Processed " << processedCount << " orders" << std::flush;
        }

    } catch (const std::exception& e) {
        res.status = 500;
        std::cerr << "Exception in server_batch: " << e.what() << std::endl;
        res.set_content(std::format(R"({{"error":"Engine error during batch processing: {}"}})", e.what()), "application/json");
    } catch (...) {
        res.status = 500;
        res.set_content(R"({"error":"Unknown internal server error during batch processing."})", "application/json");
    }
}

int main(int argc, char* argv[]) {
    // Parse port from command line argument, default to 6060
    int port = 6060;
    if (argc > 1) {
        try {
            port = std::stoi(argv[1]);
            if (port < 1024 || port > 65535) {
                std::cerr << "Port must be between 1024 and 65535, using default 6060\n";
                port = 6060;
            }
        } catch (const std::exception& e) {
            std::cerr << "Invalid port argument, using default 6060\n";
        }
    }

    // we currently access "MyMap" in all functions, which we know may run concurrently. This can be a race condition (trade + cancel at the same time).
    httplib::Server svr;

    svr.Post("/trade", server_trade);
    svr.Post("/cancel", server_cancel);
    svr.Get("/status", server_status);
    svr.Post("/reset", server_reset);
    svr.Post("/batch", server_batch);

    std::cout << "C++ server listening on http://localhost:" << port << "\n" << std::flush;
    svr.listen("0.0.0.0", port);
}

