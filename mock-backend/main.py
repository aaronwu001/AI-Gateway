from fastapi import FastAPI, Request
import time
import asyncio
import os

app = FastAPI()

# 模擬環境變數 (服務名稱)
SERVICE_NAME = os.getenv("SERVICE_NAME", "Mock-AI-Service")

@app.get("/")
async def root():
    return {"message": f"Hello from {SERVICE_NAME}"}

@app.get("/api/v1/gpt4")
async def mock_gpt4():
    """模擬一個很慢的大語言模型 (LLM)"""
    print(f"[{SERVICE_NAME}] 收到 GPT-4 請求... 開始思考...")
    # 模擬 GPU 運算延遲 (2秒)
    await asyncio.sleep(2)
    return {
        "model": "gpt-4-turbo",
        "response": "這是一個模擬的 AI 回應。我思考了 2 秒鐘才產生這句話。",
        "usage": {"prompt_tokens": 10, "completion_tokens": 20}
    }

@app.get("/api/v1/vision")
async def mock_vision():
    """模擬一個較快的影像辨識模型"""
    print(f"[{SERVICE_NAME}] 收到圖片辨識請求...")
    # 模擬較快的運算 (0.5秒)
    await asyncio.sleep(0.5)
    return {
        "model": "resnet-50",
        "tags": ["cat", "animal", "cute"],
        "confidence": 0.98
    }

if __name__ == "__main__":
    import uvicorn
    # 監聽 0.0.0.0 讓 Docker 外部可以連線
    uvicorn.run(app, host="0.0.0.0", port=5001)