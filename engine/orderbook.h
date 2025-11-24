#include <map>
#include <unordered_map>
#include <list>
#include <vector>
#include <cstdint>
#include <memory>
#include <iterator>

enum class OrderType
{
    GoodTillCancel,
    FillAndKill
};

enum class Side
{
    Buy,
    Sell
};

using Price = std::int32_t;
using Quantity = std::uint32_t;
using OrderId = std::uint64_t;

struct LevelInfo
{
    Price price_;
    Quantity quantity_;
};

using LevelInfos = std::vector<LevelInfo>;

class OrderBookLevelInfo
{
public:
    OrderBookLevelInfo(const LevelInfos &asks, const LevelInfos &bids);
        const LevelInfos& GetBids() const { return bids_; }
        const LevelInfos& GetAsks() const { return asks_; }

private:
    LevelInfos bids_;
    LevelInfos asks_;
};

class Order
{
public:
    Order(OrderType orderType, Side side, Price price, Quantity quantity, OrderId orderId);
        OrderId GetOrderId() const { return orderId_; }
        Side GetSide() const { return side_; }
        Price GetPrice() const { return price_; }
        OrderType GetOrderType() const { return orderType_; }
        Quantity GetInitialQuantity() const { return initialQuantity_; }
        Quantity GetRemainingQuantity() const { return remainingQuantity_; }
        Quantity FilledQuantity() const { return GetInitialQuantity() - GetRemainingQuantity();}
        bool IsFilled() const { return GetRemainingQuantity() == 0;}

    void Fill();

private:
    OrderType orderType_;
    OrderId orderId_;
    Price price_;
    Side side_;
    Quantity initialQuantity_;
    Quantity remainingQuantity_;
};

using OrderPointer = std::shared_ptr<Order>;
using OrderPointers = std::list<OrderPointer>;

class OrderModify{
    public:
        OrderModify(OrderId orderId, Side side, Price price, Quantity quantity);
        OrderId GetOrderId() const {return orderId_;}
        Price GetPrice() const {return price_;}
        Side GetSide() const {return side_;}
        Quantity GetQuantity() const {return quantity_;}

        OrderPointer ToOrderPointer(OrderType type) const;
    private:
        OrderId orderId_;
        Side side_;
        Price price_;
        Quantity quantity_;
};

struct TradeInfo{
    OrderId orderid_;
    Price price_;
    Quantity quantity_;
};

class Trade{
    public:
        Trade(const TradeInfo& bidTrade, const TradeInfo& askTrade);
        const TradeInfo& GetBidTrade(){return bidTrade_;}
        const TradeInfo& GetAskTrade(){return askTrade_;}
    private:
        TradeInfo bidTrade_;
        TradeInfo askTrade_;
};

using Trades = std::vector<Trade>;

class OrderBook{
    private:    
        struct OrderEntry{
                OrderPointer order_ { nullptr };
                OrderPointers::iterator location_;
            };
        std::map<Price, OrderPointers, std::greater<Price>> bids_;
        std::map<Price, OrderPointers, std::less<Price>> asks_;

        std::unordered_map<OrderId, OrderEntry> orders_;

        bool CanMatch(Side side, Price price) const;
        Trades MatchOrders();
    
    public:
        Trades AddOrder(OrderPointer order); 
        Trades MatchOrder(OrderModify order);
        std::size_t Size() const;

        OrderBookLevelInfo GetOrderInfos() const;

};

OrderType setType(std::string type);

Side setSide(std::string side);