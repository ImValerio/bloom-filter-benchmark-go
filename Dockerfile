
# Use the official Redis image from Docker Hub as the base image
FROM redis:latest

# Expose the default Redis port
EXPOSE 6379

# Optional: Add custom configuration (if you have a custom redis.conf file)
# COPY redis.conf /usr/local/etc/redis/redis.conf
# CMD ["redis-server", "/usr/local/etc/redis/redis.conf"]

# Default command to start Redis
CMD ["redis-server"]