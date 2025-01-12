#include <fstream>
#include <iostream>
#include <thread>

template <typename D>
void SimulateWork(D duration, size_t memory) {
    std::this_thread::sleep_for(duration);

    char* arr = new char[memory];
    for (size_t i = 0; i < memory; ++i) {
        arr[i] = 1;
    }
};

int main() {
    using namespace std::chrono_literals;

    for (int i = 1; i <= 5; ++i) {
        std::string cmd, args;
        std::cin >> cmd >> args;

        if (cmd == "echo") {
            SimulateWork(400ms, 32 * 1024 * 1024);

            // stoi used specifically because it throws
            std::cout << std::stoi(args) << std::endl;
        } else if (cmd == "field") {
            SimulateWork(2s, 100 * 1024 * 1024);

            std::ofstream f(args);
            f <<
                "10 10\n"
                "1 h 0 0\n"
                "2 h 0 2\n"
                "3 h 0 4\n"
                "4 h 0 6\n";
            f.close();

            std::cout << "ok" << std::endl;

            SimulateWork(500ms, 300 * 1024 * 1024);
        }
    }

    return 0;
}
