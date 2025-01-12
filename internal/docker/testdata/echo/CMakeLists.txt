cmake_minimum_required(VERSION 3.10)

project(echo)

set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -O3")

add_executable(echo main.cpp)
