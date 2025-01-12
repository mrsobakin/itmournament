#include <iostream>

const size_t kSize = 128 * 1024 * 1024;

int main() {
    char* arr = new char[kSize];

    for (size_t i = 0; i < kSize; ++i) {
        arr[i] = 1;
    }

    std::cout << "ok" << std::endl;

    return 0;
}
