#include "httplib.h"
#include <iostream>
#include <string>

int main() {
    httplib::Server svr;

    svr.Post("/run", [](const httplib::Request& req, httplib::Response& res) {
        try {
            // Read query parameters (?action=BUY&amount=10)
            std::string action = req.get_param_value("action");
            std::string amount = req.get_param_value("amount");

            std::cout << "[C++] Received request:" << std::endl;
            std::cout << "  action = " << action << std::endl;
            std::cout << "  amount = " << amount << std::endl;

            // Respond back
            res.set_content("OK", "text/plain");
        }
        catch (const std::exception& e) {
            res.status = 400;
            res.set_content("Bad Request", "text/plain");
        }
    });

    std::cout << "C++ server listening on http://localhost:6060/run\n";
    svr.listen("0.0.0.0", 6060);
}
