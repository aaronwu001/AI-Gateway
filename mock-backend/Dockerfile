# Use a lightweight Python image
FROM python:3.9-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy source code
COPY . .

# Expose port (matches main.py)
EXPOSE 5001

# Startup command
CMD ["python", "main.py"]