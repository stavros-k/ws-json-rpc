const ws = new WebSocket("ws://localhost:8080/ws");

ws.onopen = () => {
  console.log("Connected to WebSocket");

  ws.send(
    JSON.stringify({ method: "echo", params: { message: "hello" }, id: 1 })
  );

  ws.send(
    JSON.stringify({
      method: "subscribe",
      params: { event: "user.update" },
      id: 2,
    })
  );
};

function onmessage(event) {
  const data = JSON.parse(event.data);
  console.log("Message from server:", data);
}

ws.onmessage = onmessage;

ws.onerror = (error) => {
  console.error("WebSocket error:", error);
};

ws.onclose = (event) => {
  console.log("WebSocket closed:", event.code, event.reason);
};
