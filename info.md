# What is an order book?

Order books are data structures used in finance to keep track of all b uy and sell orders for assets at a different price. It has two sides (buy and selL).

Entries in the order book show:
- Who wants to buy
- Who wants to sell
- Price of ask
- How much Volume

#### Levels

A "Level" of an order book refers to all orders at some same "level" price.

In an order book, the "best bid" is the highest buy price, and "best ask" is the lowest sell price.
The difference between those two values is the "spread".

#### spread

The spread measures liquidty. small spread means a high liquidty, and high spread means a low liquidty (buy and ask is too far apart, transactions affect price / volatility).

### Order Types

- Limit Order (Buy/Sell at a specific price or better).
- Market Order (Buy/Sell at the best availible price right now).

Market orders are instant, and therefore consume liquidty, whereas limit orders (conservative) provide liquidty.

The software developed here is the OrderBook within a matching engine, that essentially processes orders.
