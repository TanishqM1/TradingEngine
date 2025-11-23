#include <iostream>
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
        const LevelInfos& GetAsks() const { return asks_;}
    
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
using OrderPointers = std::list<OrderPointer>;

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
        return std::make_shared<Order>(type, GetOrderId(), GetSide(), GetPrice(), GetQuantity());
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
                
                // NEED TO UNDERSTAND THIS BETTER.
                OrderPointers::iterator iterator;

                if (order->GetSide() == Side::Buy){
                    auto& orders = bids_[order->GetPrice()];
                    orders.push_back(order);
                    iterator = std::next(orders.begin(), orders.size()-1);
                }else{
                    auto& orders = asks_[order->GetPrice()];
                    orders.push_back(order);
                    iterator = std::next(orders.begin(), orders.size()-1);
                }

                orders_.insert({order->GetOrderId(), OrderEntry{ order, iterator}});
                return MatchOrders();
            }

            void CancelOrder(OrderId orderId){
            if (!orders_.contains(orderId)){
                return;
            }
            const auto& [order, orderIterator] = orders_.at(orderId);
            orders_.erase(orderId);

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

                const auto& [existingOrder, _] = orders_.at(order.GetOrderId());
                CancelOrder(order.GetOrderId());
                return AddOrder(order.ToOrderPointer(existingOrder->GetOrderType()));
            }
};

int main(){

    return 0;
}