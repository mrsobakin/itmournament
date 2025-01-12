#include <fstream>
#include <iostream>
#include <thread>

int main() {
    using namespace std::chrono_literals;

    std::ofstream f("/tmp/file.txt");
    f << "test data";
    f.close();

    std::this_thread::sleep_for(2s);

    std::cout << "ok" << std::endl;

    return 0;
}
