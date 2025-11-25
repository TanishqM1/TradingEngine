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

        const TradeInfo& GetBidTrade(){return bidTrade_;}
        const TradeInfo& GetAskTrade(){return askTrade_;}

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
          }

        // We also need a Match() function that runs when a match actually occurs. 
        Trades MatchOrders(){
            Trades trades;
            // .reserve() in cpp modifies the capacity of an object (how much memory is allocated), to prevent unneeded alloation down the line.
            trades.reserve(orders_.size());

            while (true){
                if (bids_.empty() || asks_.empty()){ break;}
                
                // this is `Structured Binding Declaration`. asks_.begin() returns an iterator to the first pair, but since we dereference it, it returns the 
                // PRICE (askprice/bidprice, the key) and the OrderPointers (list of ALL orders at this price) and stores it into the variables as needed.
                auto&[askPrice, asks] = *asks_.begin();
                auto&[bidPrice, bids] = *bids_.begin();
            
                // if no matches can be done (bid too low / ask too high), we return.
                if (bidPrice < askPrice){ break; }

                while (!bids.empty() && !asks.empty()){
                    auto& bid = bids.front();
                    auto& ask = asks.front();

                    /*
                    bid/ask are OrderPointers. given the pointer to an order, find out the minimum quantity we need to fill (i.e the lesser of the two possible quantities).
                    so we get the quantity of the bid() order, and the ask() order.
                    the "->" operatior below dereferences the pointer, so we can use GetRemainingQuantity() on the actual underlying object. It's logically as such:
                    bid-> GetRemainingQuantity() === (*bid).GetRemainingQuantity().
                    */

                    Quantity quantity = std::min(bid-> GetRemainingQuantity(), ask-> GetRemainingQuantity());
                    
                    // call fill() on the orders.
                    bid->Fill(quantity);
                    ask->Fill(quantity);
                    
                    // after filling, if it is compeltely filled (not partially), we can REMOVE it from our orderbook.
                    if (bid->IsFilled()){
                        bids.pop_front();
                        orders_.erase(bid->GetOrderId());
                    }
                    if (ask->IsFilled()){
                        asks.pop_front();
                        orders_.erase(ask->GetOrderId());
                    }

                    if (bids.empty()){ bids_.erase(bidPrice);}
                    if (asks.empty()){ asks_.erase(askPrice);}

                    // add to our log of the trade (requires two TradeInfo objects).
                    trades.push_back(Trade{
                        TradeInfo{ bid->GetOrderId(), bid->GetPrice(), quantity},
                        TradeInfo{ ask->GetOrderId(), ask->GetPrice(), quantity}
                    });

                }
            }
            // we've taken care of cleaning up bid/ask references after an order fill in the case it was partially filled, it's no longer availible, and we've also logged the trade that has happend.
            // But, we also need to update the bid and ask at the current price. This is important for FUTURE orders, but ALSO if the order was partially filled, as we need to re-run the engine to continue seeing
            // if it can be filled at a NEW price.

            // essentially, we need to re-set the best bid and ask price(s), after our current order has gone through.
            if (!bids_.empty()){
                auto& [_, bids] = *bids_.begin();
                auto& order = bids.front();
                // if the type is fill and kill, we ALWAYS want to remove it from the orderbook right away.
                if (order-> GetOrderType() == OrderType::FillAndKill){
                    CancelOrder(order->GetOrderId());
                }
            }

            if (!asks_.empty()){
                auto& [_, asks] = *asks_.begin();
                auto& order = asks.front();
                if (order-> GetOrderType() == OrderType::FillAndKill){
                    CancelOrder(order-> GetOrderId());
                }
            }
            return trades; // finally, we can return the trades we JUST accomplished.   
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

std::unordered_map<string, Orderbook> MyMap;

OrderType parse_ordertype(string type){
    if (type == "GTC"){return OrderType::GoodTillCancel;}
    else{return OrderType::FillAndKill;}
}

Side parse_side(string side){
    if (side=="BUY"){return Side::Buy;}
    else{return Side::Sell;}
}

Price parse_price(string price){
    int temp_int = stoi(price);
    int32_t res = static_cast<int32_t>(temp_int);
    return res;
}

Quantity parse_quantity(string quantity){
    int temp_quantity = stoi(quantity);
    uint32_t res = static_cast<uint32_t>(temp_quantity);
    return res;
}
OrderId parse_id(string id){
    int temp_id = stoi(id);
    uint64_t res = static_cast<uint64_t>(temp_id);
    return res;
}


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

    
        Orderbook& book = MyMap[s_book];
    
        // GOOG.AddOrder(std::make_shared<Order>(OrderType::GoodTillCancel, Side::Buy, 100, 10, orderId));
        book.AddOrder(std::make_shared<Order>(type, side, price, quantity, id));
        res.status = 200; // or httplib::StatusCode::OK_200
        res.set_content("{\"message\": \"Order placed successfully\"}", "application/json");
        cout << "\n " << book.Size();
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
            res.set_content("{\"message\": \"Order ID not found\"}", "application/json");
        }
    }catch(...){
        res.status = 500;
        res.set_content(R"({"error":"Unknown internal server error."})", "application/json");
        cout << "\n There was an error";
    }
}

void print_state(const httplib::Request& req, httplib::Response& res){
    // prints all levels of each orderbook.
}

int main() {
    httplib::Server svr;
    Orderbook GOOG;

    svr.Post("/trade", server_trade);
    svr.Post("/cancel", server_cancel);

    std::cout << "C++ server listening on http://localhost:6060/run\n";
    svr.listen("0.0.0.0", 6060);

    
}

