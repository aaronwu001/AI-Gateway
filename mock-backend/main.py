from fastapi import FastAPI, Request
import asyncio
import os

app = FastAPI()

# Simulated Environment Variable
SERVICE_NAME = os.getenv("SERVICE_NAME", "Mock-AI-Service")

@app.get("/")
async def root():
    return {"message": f"Hello from {SERVICE_NAME}"}

@app.api_route("/api/v1/gpt4", methods=["GET", "POST"])
async def mock_gpt4(request: Request):
    """Simulates a slow LLM (Supports POST for tests, GET for browser)"""
    
    if request.method == "POST":
        try:
            body = await request.json()
            print(f"[{SERVICE_NAME}] Received GPT-4 POST Request: {body}")
        except:
            print(f"[{SERVICE_NAME}] Received GPT-4 POST Request (Empty Body)")
    else:
        print(f"[{SERVICE_NAME}] Received GPT-4 GET Request")

    # Simulate Latency (0.1s is enough for testing)
    await asyncio.sleep(0.1) 
    
    return {
        "model": "gpt-4-turbo",
        "choices": [{
            "message": {
                "role": "assistant",
                "content": "This is a mock response from the Docker container."
            },
            "finish_reason": "stop"
        }],
        "usage": {"prompt_tokens": 10, "completion_tokens": 20, "total_tokens": 30}
    }

@app.api_route("/api/v1/vision", methods=["GET", "POST"])
async def mock_vision(request: Request):
    """Simulates a faster Vision Model"""
    print(f"[{SERVICE_NAME}] Received Vision Request...")
    
    await asyncio.sleep(0.1)
    
    return {
        "model": "resnet-50",
        "tags": ["cat", "animal", "cute"],
        "confidence": 0.98
    }

if __name__ == "__main__":
    import uvicorn
    # Listen on 0.0.0.0 to allow access from outside Docker
    uvicorn.run(app, host="0.0.0.0", port=5001)