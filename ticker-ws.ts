import type { OHLCVData } from "./ticker-db";
import { getTickerDatabase } from "./ticker-db";

class BitgetTickerWS {
  private ws: WebSocket | null = null;
  private reconnectInterval: number = 5000; // 5 seconds
  private isConnecting: boolean = false;
  private isSubscribed: boolean = false;
  private db = getTickerDatabase();

  // Write buffering for performance
  private writeBuffer: OHLCVData[] = [];
  private bufferSize = 100; // Configurable buffer size
  private flushInterval = 50; // ms - time between forced flushes
  private flushTimer: ReturnType<typeof setTimeout> | null = null;

  // Performance metrics
  private metrics = {
    totalProcessed: 0,
    totalInserted: 0,
    avgInsertTime: 0,
    bufferFlushes: 0,
    lastFlushTime: Date.now(),
  };

  constructor() {
    // Constructor for WebSocket functionality
    this.startPeriodicFlush();
  }

  // Start WebSocket connection and subscription
  async startWebSocketStream() {
    if (
      this.isConnecting ||
      (this.ws && this.ws.readyState === WebSocket.OPEN)
    ) {
      console.log("🔌 WebSocket already connected or connecting");
      return;
    }

    await this.connectWebSocket();
  }

  // Connect to Bitget WebSocket stream
  private async connectWebSocket() {
    this.isConnecting = true;

    try {
      this.ws = new WebSocket(
        "wss://stream.bitget.com/public/v1/stream?terminalType=1"
      );

      this.ws.onopen = () => {
        console.log("✅ WebSocket connected successfully");
        this.isConnecting = false;
        this.subscribeToTickers();
      };

      this.ws.onmessage = (event) => {
        this.handleWebSocketMessage(event.data);
      };

      this.ws.onerror = (error) => {
        console.error("❌ WebSocket error:", error);
        this.isConnecting = false;
      };

      this.ws.onclose = (event) => {
        this.isConnecting = false;
        this.isSubscribed = false;
        this.ws = null;

        // Attempt to reconnect after delay
        if (!event.wasClean) {
          console.log(
            `🔄 Attempting to reconnect in ${
              this.reconnectInterval / 1000
            } seconds...`
          );
          setTimeout(() => {
            this.connectWebSocket();
          }, this.reconnectInterval);
        }
      };
    } catch (error) {
      console.error("❌ Failed to connect to WebSocket:", error);
      this.isConnecting = false;

      // Retry connection
      setTimeout(() => {
        this.connectWebSocket();
      }, this.reconnectInterval);
    }
  }

  // Subscribe to tickers channels
  private subscribeToTickers() {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      console.error("❌ WebSocket not ready for subscription");
      return;
    }

    try {
      // Subscribe to margin tickers
      const marginSubscribe = {
        op: "subscribe",
        args: [
          {
            instType: "mc",
            instId: "default",
            channel: "tickersGroup",
            params: { update: "1" },
          },
        ],
      };

      const subscribeMessage = JSON.stringify(marginSubscribe);
      console.log("📡 Sending subscription:", subscribeMessage);
      this.ws.send(subscribeMessage);
      console.log("✅ Subscription sent successfully");

      this.isSubscribed = true;
    } catch (error) {
      console.error("❌ Failed to send subscription messages:", error);
    }
  }

  // Handle incoming WebSocket messages
  private handleWebSocketMessage(data: string) {
    try {
      const message = JSON.parse(data);

      if (message.event === "subscribe") {
        console.log("✅ Subscription confirmation received");
        return;
      }

      if (message.code === "0" && message.msg === "success") {
        console.log("✅ Success response received");
        return;
      }

      // Handle ticker data
      if (message.data && Array.isArray(message.data)) {
        this.processTickerData(message.data);
      }
    } catch (error) {
      console.error("❌ Failed to parse WebSocket message:", error);
      console.log("Raw message data:", data);
    }
  }

  // Process ticker data from WebSocket
  private processTickerData(tickerData: any[]) {
    try {
      const startTime = performance.now();
      const ohlcvDataList: OHLCVData[] = [];

      for (const ticker of tickerData) {
        // Bitget margin ticker uses different field names
        if (ticker.sC && ticker.p) {
          const ohlcvData: OHLCVData = {
            symbol: ticker.sC, // Symbol (e.g., BTCUSDT)
            open: ticker.oB ? parseFloat(ticker.oB) : parseFloat(ticker.p), // openBid or current price
            high: ticker.h ? parseFloat(ticker.h) : parseFloat(ticker.p), // high
            low: ticker.l ? parseFloat(ticker.l) : parseFloat(ticker.p), // low
            close: parseFloat(ticker.p), // current price
            volume: ticker.qV ? parseFloat(ticker.qV) : 0, // quoteVolume
            timestamp: ticker.t ? parseInt(ticker.t) : Date.now(), // timestamp
            interval: "24h",
            vwap: ticker.iP ? parseFloat(ticker.iP) : undefined, // indexPrice
          };

          ohlcvDataList.push(ohlcvData);
        }
      }

      if (ohlcvDataList.length > 0) {
        this.metrics.totalProcessed += ohlcvDataList.length;

        // Use buffered writing instead of direct insertion
        this.writeBuffer.push(...ohlcvDataList);

        if (this.writeBuffer.length >= this.bufferSize) {
          this.flushBuffer();
        } else {
          this.scheduleFlush();
        }

        // Update performance metrics
        const endTime = performance.now();
        const processTime = endTime - startTime;
        this.metrics.avgInsertTime =
          (this.metrics.avgInsertTime + processTime) / 2;

        // Log metrics periodically
        if (
          this.metrics.totalProcessed % 10000 === 0 &&
          this.metrics.totalProcessed > 0
        ) {
          console.log(`📊 Performance Metrics:`, {
            totalProcessed: this.metrics.totalProcessed,
            totalInserted: this.metrics.totalInserted,
            avgInsertTime: `${this.metrics.avgInsertTime.toFixed(2)}ms`,
            bufferFlushes: this.metrics.bufferFlushes,
            bufferSize: this.writeBuffer.length,
          });
        }
      }
    } catch (error) {
      console.error("❌ Failed to process ticker data:", error);
    }
  }

  // Schedule buffer flush if not already scheduled
  private scheduleFlush(): void {
    if (this.flushTimer) return;

    this.flushTimer = setTimeout(() => {
      this.flushBuffer();
    }, this.flushInterval);
  }

  // Flush the write buffer to database
  private flushBuffer(): void {
    if (this.writeBuffer.length === 0) return;

    const batch = [...this.writeBuffer];
    this.writeBuffer = [];

    if (this.flushTimer) {
      clearTimeout(this.flushTimer);
      this.flushTimer = null;
    }

    this.db.bulkInsertOHLCV(batch);
    this.metrics.totalInserted += batch.length;
    this.metrics.bufferFlushes++;
    this.metrics.lastFlushTime = Date.now();
  }

  // Start periodic flush to prevent data buildup
  private startPeriodicFlush(): void {
    setInterval(() => {
      if (this.writeBuffer.length > 0) {
        this.flushBuffer();
      }
    }, 1000); // Force flush every second if buffer has data
  }

  // Stop WebSocket connection
  async stopWebSocketStream() {
    if (this.ws) {
      console.log("🛑 Stopping WebSocket stream...");

      // Flush any remaining buffer data before closing
      this.flushBuffer();

      this.ws.close(1000, "Client disconnect");
      this.ws = null;
      this.isSubscribed = false;
      this.isConnecting = false;
    }
  }

  // Get WebSocket connection status
  getWebSocketStatus(): {
    connected: boolean;
    subscribed: boolean;
    connecting: boolean;
  } {
    return {
      connected: this.ws?.readyState === WebSocket.OPEN,
      subscribed: this.isSubscribed,
      connecting: this.isConnecting,
    };
  }

  // Close WebSocket connection
  async close() {
    // Stop WebSocket stream
    await this.stopWebSocketStream();
    console.log("🛑 PriceTracker closed successfully");
  }
}

export default BitgetTickerWS;
