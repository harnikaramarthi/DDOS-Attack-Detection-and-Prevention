# Create a test payload
$payload = "A" * 15  # Creates a string of 15 bytes

# Test endpoint
$uri = "http://localhost:3000"

# Send request with large payload
try {
    $response = Invoke-WebRequest -Uri $uri -Method POST -Body $payload -ContentType "text/plain"
    Write-Host "Response Status Code: $($response.StatusCode)"
} catch {
    Write-Host "Error Status Code: $($_.Exception.Response.StatusCode)"
    Write-Host "Error Message: $($_.Exception.Message)"
}