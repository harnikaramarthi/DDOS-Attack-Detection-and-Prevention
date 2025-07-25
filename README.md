# Block Portal

A reverse proxy server that protects your applications by limiting and blocking suspicious IP addresses.

## Features

- Rate limiting per IP address
- Reverse proxy functionality
- Configurable request limits
- Easy to set up and test

## How to Run

1. Install Golang: https://go.dev/
2. First, make sure you are in the root directory of the project:
   ```bash
   cd blockportal
   ```

### Terminal 1 - Test Server
```bash
# Navigate to the simple directory
cd simple

# Run the test server (will run on port 8081)
go run main.go
```

### Terminal 2 - Proxy Server
```bash
# Build the proxy
go build -o blockportal.exe

# Run with default settings (1 request per second)
./blockportal -url localhost:8081

# Or customize the settings
./blockportal -url localhost:8081 -limit 5 -port 3001
```

## Testing the Rate Limiting

Here's how to test the rate limiting functionality:

### Method 1: Using a Web Browser

1. Start both servers:
   - Test server on port 8081
   - Proxy server on port 3000
2. Open your browser and visit: http://localhost:3000
3. Quickly refresh the page multiple times (F5 or Ctrl+R)
4. After a few refreshes, you should see the "Rate limit exceeded" message
5. Wait for 1-2 seconds, then try again - it should work

### Method 2: Using Command Line (Windows)

1. Open Command Prompt (cmd) or PowerShell
2. Run these commands to test:

```bash
# Single request (should succeed)
curl http://localhost:3000

# Multiple rapid requests (some should fail)
# Method A - Quick sequential requests
for /L %i in (1,1,5) do curl http://localhost:3000

# Method B - Parallel requests (will definitely trigger rate limit)
for /L %i in (1,1,5) do start curl http://localhost:3000
```

### Method 3: Using PowerShell

```powershell
# Multiple requests with timing
1..5 | ForEach-Object { 
    $response = Invoke-WebRequest -Uri "http://localhost:3000"
    Write-Host "Request $_ : $($response.StatusCode)"
    Start-Sleep -Milliseconds 200
}
```

### Expected Results:

- First few requests: Status 200 OK with "Hello, world!" message
- After rate limit: Status 429 with "Rate limit exceeded" message
- After waiting: Requests work again

### Adjusting Rate Limits

To test different rate limits, restart the proxy with different settings:

```bash
# Allow 5 requests per second
./blockportal -url localhost:8081 -limit 5

# Very strict: only 1 request every 2 seconds
./blockportal -url localhost:8081 -limit 1 -port 3000
```

## Advanced Security Features

### DDoS Protection Testing

**Prerequisites:**
1. Open two terminals/command prompts
2. Navigate to the blockportal directory in both:
   ```bash
   cd c:\Users\gudde\desktop projects\blockportal_main\blockportal\blockportal
   ```

**Testing Steps:**
1. In Terminal 1 - Start test server:
   ```bash
   cd simple
   go run main.go
   ```

2. In Terminal 2 - Start proxy server:
   ```bash
   # Build and run proxy with security settings
   go build
   ./blockportal -url localhost:8081 -limit 5 -maxsize 1048576
   ```

3. In Terminal 3 (new Command Prompt/PowerShell) - Run tests:
   ```bash
   cd c:\Users\gudde\desktop projects\blockportal_main\blockportal\blockportal

   # Test Pattern Detection (Command Prompt)
   for /L %i in (1,1,15) do curl http://localhost:3000/same-path

   # OR PowerShell alternative
   1..15 | ForEach-Object { curl http://localhost:3000/same-path }
   ```

Expected Results:
- First few requests: Success (200 OK)
- After ~10 requests: "suspicious pattern detected"
- Then: "IP blacklisted" message

2. Test Pattern Detection:
```bash
# Rapid identical requests (should trigger pattern detection)
for /L %i in (1,1,15) do curl http://localhost:3000/same-path

# You should see "suspicious pattern detected" after several requests
```

3. Test Payload Size Limit:
```bash
# Create a large file
fsutil file createnew test.dat 2097152

# Try to upload (should fail)
curl -X POST -d @test.dat http://localhost:3000
```

4. Test IP Blacklisting:
- After multiple violations, your IP will be blacklisted
- You'll see "IP blacklisted" for 5 minutes
- Try different endpoints during blacklist period

### Security Features

1. Rate Limiting:
   - Configurable requests per second
   - Token bucket algorithm
   
2. Pattern Detection:
   - Monitors repetitive request patterns
   - Blocks potential scanning attempts
   
3. Payload Protection:
   - Configurable maximum request size
   - Prevents memory exhaustion attacks
   
4. IP Blacklisting:
   - Temporary bans for suspicious IPs
   - Automatic unbanning after timeout

## Security Testing Guide

### 1. Pattern Detection Testing (Scan Attack Simulation)

Simulates an attacker scanning your website for vulnerabilities.

```bash
# Terminal 1: Start servers
cd simple
go run main.go

# Terminal 2: Start proxy with strict settings
./blockportal -url localhost:8081 -limit 10

# Terminal 3: Simulate scan attack
# Method 1: Quick path scanning
for %x in (admin login wp-admin phpinfo.php config.php) do curl http://localhost:3000/%x

# Method 2: Aggressive scanning (will trigger pattern detection)
for /L %i in (1,1,20) do (
  curl http://localhost:3000/admin
  curl http://localhost:3000/wp-admin
  curl http://localhost:3000/config
)
```

Expected Results:
- First few requests: Normal response (200 OK)
- After ~10 similar requests: "suspicious pattern detected"
- IP gets blacklisted after continued attempts

### 2. Payload Protection Testing (Memory Attack Simulation)

Simulates attempts to overwhelm the server with large requests.

```bash
# Create test files of different sizes
fsutil file createnew small.dat 512000    # 500KB
fsutil file createnew medium.dat 1048577  # Just over 1MB
fsutil file createnew large.dat 5242880   # 5MB

# Test with different file sizes
curl -X POST -d @small.dat http://localhost:3000/upload   # Should succeed
curl -X POST -d @medium.dat http://localhost:3000/upload  # Should fail
curl -X POST -d @large.dat http://localhost:3000/upload   # Should fail

# Rapid large payload testing
for /L %i in (1,1,5) do start curl -X POST -d @medium.dat http://localhost:3000/upload
```

Expected Results:
- Small payloads (<1MB): Accepted
- Large payloads (>1MB): "request too large"
- Multiple large payloads: IP gets blacklisted

### 3. IP Blacklisting Testing (Brute Force Simulation)

Simulates a brute force attack attempt.

```bash
# Create a test script (save as test_auth.ps1)
$urls = @(
    "http://localhost:3000/login?user=admin&pass=123",
    "http://localhost:3000/login?user=admin&pass=password",
    "http://localhost:3000/login?user=root&pass=toor"
)

foreach ($i in 1..20) {
    $url = $urls | Get-Random
    $response = try {
        Invoke-WebRequest -Uri $url -Method GET
        "Request $i : $($response.StatusCode)"
    } catch {
        "Request $i : $($_.Exception.Response.StatusCode) - $($_.Exception.Response.StatusDescription)"
    }
    Start-Sleep -Milliseconds 100
}
```

Run the test:
```bash
powershell -File test_auth.ps1
```

Expected Results:
1. Initial phase:
   - Requests go through with rate limiting
   - See pattern detection warnings
2. Middle phase:
   - Some requests get blocked
   - "suspicious pattern detected"
3. Final phase:
   - IP gets blacklisted
   - All requests return "IP blacklisted"
   - Blacklist remains for 5 minutes

### How to Identify Real Attacks

Monitor these patterns that indicate potential attacks:

1. Pattern-based Attacks:
   - Many requests to sensitive paths (/admin, /wp-admin, etc.)
   - Sequential scanning of common vulnerabilities
   - High frequency of identical requests

2. Payload-based Attacks:
   - Multiple large POST/PUT requests
   - Suspicious content types
   - Malformed request bodies

3. Brute Force Attacks:
   - Rapid login attempts
   - Sequential or dictionary-based parameter variations
   - Distributed requests from multiple IPs

### Monitoring Attack Attempts

Run with verbose logging:
```bash
./blockportal -url localhost:8081 -limit 5 -verbose

# Watch the logs for patterns like:
# - "Pattern detection triggered for IP: x.x.x.x"
# - "Large payload detected: SIZE bytes from IP: x.x.x.x"
# - "IP Blacklisted: x.x.x.x - Multiple violations"
```

## Checking Results

### Where to See Results:

1. **Terminal Output**:
   - Proxy server terminal will show logs like:
     ```
     "Pattern detection triggered for IP: x.x.x.x"
     "Rate limit exceeded for IP: x.x.x.x"
     "IP Blacklisted: x.x.x.x"
     ```

2. **Browser**:
   - Visit http://localhost:3000
   - You'll see these responses directly in the browser:
     - Success: "Hello, world!"
     - Rate limit: "Rate limit exceeded"
     - Pattern detection: "suspicious pattern detected"
     - Blacklist: "IP blacklisted"

3. **Command Line Results**:
   - When using curl commands, results appear in the same terminal
   - HTTP status codes and response messages are displayed immediately
   - Example output:
     ```
     HTTP/1.1 200 OK         <- Successful request
     HTTP/1.1 429 Too Many   <- Rate limit exceeded
     HTTP/1.1 403 Forbidden  <- Pattern detected/Blacklisted
     ```

Note: The most detailed information appears in the proxy server terminal, so keep an eye on both the terminal and browser/curl responses while testing.

## How It Works

The proxy server:
1. Receives incoming requests
2. Tracks requests per IP address
3. Blocks IPs that exceed the configured rate limit
4. Forwards valid requests to your application

This protects your application from:
- DDoS attacks
- Brute force attempts
- Aggressive scrapers

## Troubleshooting

If you get a "port already in use" error:
1. Find the process using the port:
   ```bash
   # On Windows
   netstat -ano | findstr :8081
   ```
2. Kill the process:
   ```bash
   # On Windows (replace PID with the process ID from above)
   taskkill /PID <PID> /F
   ```
3. Or simply change the port in main.go to another number (e.g., 8082)
