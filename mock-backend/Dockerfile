# 使用輕量級 Python 映像檔
FROM python:3.9-slim

WORKDIR /app

# 安裝依賴
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# 複製程式碼
COPY . .

# 暴露 Port (與 main.py 一致)
EXPOSE 5001

# 啟動命令
CMD ["python", "main.py"]