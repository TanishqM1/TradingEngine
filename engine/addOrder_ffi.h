#include <stdint.h>
#include <iostream>
#include <string>

typedef void* OrderBookAddress;

void AddOrderToEngine(
    OrderBookAddress book_ptr,
    int orderType,
    int side,
    int32_t price,
    uint32_t quantity,
    uint64_t orderId
);

OrderBookAddress CreateBook(const char* name);
void DestroyBook(OrderBookAddress book_ptr);
