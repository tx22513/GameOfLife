# Conway's Game of Life - Parallel and Distributed Implementation

## Introduction

This project presents a comprehensive exploration of both parallel and distributed approaches to implementing Conway's Game of Life using Golang, a language well-suited for handling concurrent processes. The objective of this analysis is to conduct a thorough evaluation of the performance of two distinct implementations, identifying areas for improvement and optimizing code efficiency.

## Implementation Overview

### Stage 1: Parallel Implementation

#### Functionality and Design

The parallel implementation of Conway's Game of Life leverages multiple worker threads to enhance performance. Initially, a serial version was developed that operates on a single thread. This baseline version processes the entire game cycle within one thread, avoiding the complexity of parallel execution.

Transitioning to parallel processing, the focus shifted to concurrently computing different segments of the game world by employing multiple threads. Each thread is responsible for a specific portion of the computation, with the results aggregated into a cohesive outcome. A timer reporting the count of living cells every two seconds was introduced to monitor progress. Additionally, a function was developed to generate PGM images, providing a visual representation of the game state at the end of each cycle.

For real-time interaction, SDL integration allows for dynamic visualization of the game state and supports specific key presses for user commands, including generating PGM images and pausing or resuming the game.

#### Benchmark and Critical Analysis

Benchmark tests were conducted on a university lab machine with 6 cores and 12 threads. The tests demonstrated that performance improves with the addition of threads up to a certain point, beyond which the overhead of communication and synchronization diminishes the benefits. This behavior highlights the efficiency of parallel processing for medium-sized thread counts and the limitations when exceeding the physical core count of the CPU.

### Stage 2: Distributed Implementation

#### Functionality and Design

The distributed implementation extends the project to utilize multiple AWS nodes, facilitating collaborative computation and state management of the Game of Life board via RPC calls. This stage aimed to enhance scalability and leverage distributed computing to manage larger game worlds effectively.

The implementation includes a broker system to distribute tasks across servers and aggregate results, demonstrating the potential of distributed systems to handle extensive datasets with efficiency. Key performance improvements and optimization strategies, such as halo exchange, were explored to reduce communication overhead and improve overall performance.

#### Testing and Critical Analysis

Comparative tests between parallel and distributed implementations showed that the choice between these approaches depends on the dataset size. For smaller datasets, parallel processing is significantly faster due to lower communication overhead. As the dataset size increases, the advantages of distributed computing become more apparent, suggesting its suitability for larger-scale problems.

## Potential Improvements

Optimizations in the `calculateNextState` function and the adoption of bitwise operations over modulo arithmetic for boundary checking were identified as key areas for performance enhancement. Furthermore, exploring parallel distributed systems could offer scalability benefits, maximizing resource utilization across multiple computing nodes.

## Conclusion

This project underscores the importance of selecting appropriate execution strategies based on dataset size and computational requirements. Through detailed benchmarking and analysis, it was demonstrated that parallel processing offers substantial benefits for smaller datasets, while distributed computing shows promise for scalability and efficiency in handling larger datasets. Implementing optimization strategies such as halo exchange can further enhance performance, making it a viable option for large-scale implementations.
