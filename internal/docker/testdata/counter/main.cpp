#include <chrono>
#include <cstdint>
#include <iostream>

int main() {
    uint64_t counter = 0;

    auto start = std::chrono::high_resolution_clock::now();

    while (true) {
        auto now = std::chrono::high_resolution_clock::now();

        if (std::chrono::duration_cast<std::chrono::seconds>(now - start).count() >= 1.0) {
            break;
        }

        counter++;
    }

    std::cout << counter << std::endl;

    return 0;
}
