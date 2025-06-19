# Bloom filter benchmark in Go

Testing the efficiency of a Bloom filter in front of a Redis cache using:

-   go-redis/reids/v8
-   willf/bloom, bloom filter based on [MurmurHash](https://en.wikipedia.org/wiki/MurmurHash)

### Benchmark Results

| Bloom Filter | Avg Time (s) | Avg Hit Rate (%) | Speedup |
| :----------: | :----------: | :--------------: | :-----: |
|   Enabled    |    31.17     |      79.97       |  1.22x  |
|   Disabled   |    38.07     |      79.98       |  1.00x  |

#### Runs

| Run | Bloom Filter | Time (s) |   Hits | Misses | Hit Rate (%) |
| --- | :----------: | :------: | -----: | -----: | :----------: |
| 1   |   Enabled    |  32.67   | 160360 |  39640 |    80.18     |
| 2   |   Enabled    |  31.59   | 159911 |  40089 |    79.96     |
| 3   |   Enabled    |  31.59   | 159808 |  40192 |    79.90     |
| 4   |   Enabled    |  31.67   | 160099 |  39901 |    80.05     |
| 5   |   Enabled    |  30.48   | 159881 |  40119 |    79.94     |
| 6   |   Enabled    |  30.65   | 159881 |  40119 |    79.94     |
| 7   |   Enabled    |  30.96   | 159779 |  40221 |    79.89     |
| 8   |   Enabled    |  30.76   | 159706 |  40294 |    79.85     |
| 9   |   Enabled    |  30.60   | 159976 |  40024 |    79.99     |
| 10  |   Enabled    |  30.74   | 160058 |  39942 |    80.03     |
| 1   |   Disabled   |  37.71   | 159776 |  40224 |    79.89     |
| 2   |   Disabled   |  38.08   | 159925 |  40075 |    79.96     |
| 3   |   Disabled   |  37.66   | 160054 |  39946 |    80.03     |
| 4   |   Disabled   |  38.16   | 159910 |  40090 |    79.95     |
| 5   |   Disabled   |  37.39   | 159807 |  40193 |    79.90     |
| 6   |   Disabled   |  38.59   | 160052 |  39948 |    80.03     |
| 7   |   Disabled   |  38.55   | 159963 |  40037 |    79.98     |
| 8   |   Disabled   |  37.95   | 160098 |  39902 |    80.05     |
| 9   |   Disabled   |  38.27   | 160033 |  39967 |    80.02     |
| 10  |   Disabled   |  38.30   | 159981 |  40019 |    79.99     |
