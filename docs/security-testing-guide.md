# Stathera API Security Testing Guide

This guide provides instructions for testing the security features implemented in the Stathera API. It is intended for developers, QA engineers, and security testers to verify that security measures are working as expected.

## Prerequisites

- Access to a development or testing environment
- API testing tools (e.g., Postman, curl, or similar)
- Basic understanding of API security concepts
- JWT token generation capabilities
- Access to test user accounts with different permission levels

## General Testing Approach

1. **Positive Testing**: Verify that legitimate requests are processed correctly
2. **Negative Testing**: Verify that invalid or malicious requests are rejected
3. **Boundary Testing**: Test edge cases and limits
4. **Bypass Testing**: Attempt to bypass security controls

## Authentication Testing

### JWT Authentication

1. **Valid Token Test**
   - Generate a valid JWT token with correct claims
   - Send a request to a protected endpoint with the token
   - Verify that the request is processed successfully

2. **Invalid Token Test**
   - Send a request with an invalid JWT token (e.g., malformed, expired, or tampered)
   - Verify that the request is rejected with a 401 Unauthorized status

3. **Missing Token Test**
   - Send a request to a protected endpoint without a token
   - Verify that the request is rejected with a 401 Unauthorized status

4. **Token Expiration Test**
   - Generate a JWT token that is about to expire
   - Send a request with this token
   - Verify that a new token is provided in the X-New-Token header

5. **Brute Force Protection Test**
   - Send multiple requests with invalid credentials to the login endpoint
   - Verify that after a certain number of attempts, further login attempts are blocked

### API Key Authentication

1. **Valid API Key Test**
   - Send a request to a protected endpoint with a valid API key
   - Verify that the request is processed successfully

2. **Invalid API Key Test**
   - Send a request with an invalid API key
   - Verify that the request is rejected with a 401 Unauthorized status

3. **Permission Test**
   - Send a request with an API key that lacks the required permissions
   - Verify that the request is rejected with a 403 Forbidden status

## Authorization Testing

### Role-Based Access Control

1. **Admin Access Test**
   - Generate a token with admin role
   - Send a request to an admin-only endpoint
   - Verify that the request is processed successfully

2. **Regular User Access Test**
   - Generate a token with regular user role
   - Send a request to an admin-only endpoint
   - Verify that the request is rejected with a 403 Forbidden status

3. **Permission-Based Access Test**
   - Generate tokens with different permission sets
   - Send requests to endpoints requiring specific permissions
   - Verify that access is granted or denied based on permissions

### Object-Level Authorization

1. **Resource Ownership Test**
   - Generate a token for User A
   - Attempt to access a resource owned by User B
   - Verify that the request is rejected with a 403 Forbidden status

2. **Admin Override Test**
   - Generate a token with admin role
   - Attempt to access a resource owned by another user
   - Verify that the request is processed successfully

## Rate Limiting Testing

1. **IP-Based Rate Limit Test**
   - Send multiple requests from the same IP in a short period
   - Verify that after exceeding the limit, requests are rejected with a 429 Too Many Requests status
   - Verify that the Retry-After header is included in the response

2. **User-Based Rate Limit Test**
   - Send multiple authenticated requests in a short period
   - Verify that after exceeding the limit, requests are rejected with a 429 Too Many Requests status

3. **Path-Specific Rate Limit Test**
   - Send multiple requests to a specific endpoint
   - Verify that rate limits are applied based on the endpoint

4. **Rate Limit Reset Test**
   - Exceed the rate limit
   - Wait for the rate limit window to pass
   - Verify that requests are processed again

## Input Validation Testing

1. **Content Type Validation Test**
   - Send a request with an incorrect Content-Type header
   - Verify that the request is rejected with a 415 Unsupported Media Type status

2. **Input Sanitization Test**
   - Send requests with potentially malicious input (e.g., SQL injection, XSS)
   - Verify that the input is properly sanitized or rejected

3. **Parameter Validation Test**
   - Send requests with invalid parameters (e.g., negative values, too long strings)
   - Verify that the request is rejected with a 400 Bad Request status

4. **Required Field Test**
   - Send requests missing required fields
   - Verify that the request is rejected with a 400 Bad Request status

## CSRF Protection Testing

1. **CSRF Token Validation Test**
   - Send a state-changing request without a CSRF token
   - Verify that the request is rejected with a 403 Forbidden status

2. **Valid CSRF Token Test**
   - Obtain a valid CSRF token
   - Send a state-changing request with the token
   - Verify that the request is processed successfully

3. **Invalid CSRF Token Test**
   - Send a state-changing request with an invalid CSRF token
   - Verify that the request is rejected with a 403 Forbidden status

4. **CSRF Exemption Test**
   - Send a state-changing request with a valid API key but no CSRF token
   - Verify that the request is processed successfully (API key requests are exempt from CSRF protection)

## Security Headers Testing

1. **Secure Headers Presence Test**
   - Send any request to the API
   - Verify that the response includes the following headers:
     - X-Content-Type-Options: nosniff
     - X-Frame-Options: DENY
     - X-XSS-Protection: 1; mode=block
     - Referrer-Policy: strict-origin-when-cross-origin
     - Strict-Transport-Security: max-age=31536000; includeSubDomains
     - Content-Security-Policy

2. **Content Security Policy Test**
   - Examine the Content-Security-Policy header
   - Verify that it includes appropriate restrictions

## Error Handling Testing

1. **Error Response Format Test**
   - Trigger various error conditions
   - Verify that error responses follow the standard format
   - Verify that error messages do not expose sensitive information

2. **Status Code Test**
   - Trigger various error conditions
   - Verify that appropriate HTTP status codes are returned

3. **Error Logging Test**
   - Trigger various error conditions
   - Verify that errors are properly logged with appropriate context

## Injection Protection Testing

### SQL Injection Testing

1. **SQL Injection Test**
   - Send requests with SQL injection payloads in parameters
   - Verify that the requests are rejected or sanitized

### XSS Protection Testing

1. **XSS Payload Test**
   - Send requests with XSS payloads in parameters
   - Verify that the requests are rejected or sanitized

## Logging and Monitoring Testing

1. **Request Logging Test**
   - Send various requests to the API
   - Verify that requests are logged with appropriate details
   - Verify that sensitive data is not logged

2. **Security Event Logging Test**
   - Trigger security events (e.g., authentication failures, rate limit hits)
   - Verify that these events are logged with appropriate severity

3. **Request ID Tracking Test**
   - Send a request to the API
   - Verify that a request ID is generated and included in logs

## Automated Security Testing

### Using OWASP ZAP

1. **API Scanning**
   ```bash
   # Example ZAP command for API scanning
   zap-cli quick-scan --self-contained --start-options "-config api.disablekey=true" \
     --spider --ajax --scan --recursive https://api.example.com
   ```

2. **Active Scanning**
   ```bash
   # Example ZAP command for active scanning
   zap-cli active-scan --scanners all https://api.example.com
   ```

### Using API Fuzzing Tools

1. **API Fuzzing with ffuf**
   ```bash
   # Example ffuf command for API fuzzing
   ffuf -w wordlist.txt -u https://api.example.com/endpoint?param=FUZZ \
     -H "Authorization: Bearer YOUR_TOKEN" -mc 200,500
   ```

## Security Testing Checklist

Use this checklist to ensure all security features are tested:

- [ ] JWT authentication works correctly
- [ ] API key authentication works correctly
- [ ] Brute force protection is effective
- [ ] Role-based access control is enforced
- [ ] Object-level authorization is enforced
- [ ] Rate limiting is applied correctly
- [ ] Input validation rejects invalid input
- [ ] CSRF protection is effective
- [ ] Security headers are present and correct
- [ ] Error handling does not expose sensitive information
- [ ] SQL injection protection is effective
- [ ] XSS protection is effective
- [ ] Logging captures security events
- [ ] Request IDs are generated and tracked

## Reporting Security Issues

If you discover a security vulnerability during testing:

1. Document the vulnerability with clear steps to reproduce
2. Assess the severity and potential impact
3. Report the issue immediately to the security team
4. Do not disclose the vulnerability publicly until it has been addressed

## Example Test Scripts

### JWT Authentication Test

```bash
# Generate a valid JWT token
TOKEN=$(curl -s -X POST https://api.example.com/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"password"}' | jq -r '.data.token')

# Test a protected endpoint
curl -s -X GET https://api.example.com/balance \
  -H "Authorization: Bearer $TOKEN" | jq

# Test with invalid token
curl -s -X GET https://api.example.com/balance \
  -H "Authorization: Bearer invalid.token.here" | jq
```

### Rate Limiting Test

```bash
# Send multiple requests to trigger rate limiting
for i in {1..150}; do
  response=$(curl -s -w "%{http_code}" -X GET https://api.example.com/health)
  code=${response: -3}
  if [ "$code" == "429" ]; then
    echo "Rate limit triggered after $i requests"
    break
  fi
done
```

### CSRF Protection Test

```bash
# Get a session and CSRF token
SESSION_RESPONSE=$(curl -s -c cookies.txt -X POST https://api.example.com/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"password"}')

CSRF_TOKEN=$(echo $SESSION_RESPONSE | jq -r '.data.csrf_token')

# Test with valid CSRF token
curl -s -b cookies.txt -X POST https://api.example.com/transfer \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: $CSRF_TOKEN" \
  -d '{"receiver_address":"addr123","amount":10}' | jq

# Test without CSRF token
curl -s -b cookies.txt -X POST https://api.example.com/transfer \
  -H "Content-Type: application/json" \
  -d '{"receiver_address":"addr123","amount":10}' | jq
```

## Conclusion

Regular security testing is essential to maintain the security posture of the Stathera API. By following this guide, you can verify that security measures are working as expected and identify potential vulnerabilities before they can be exploited.

Remember that security testing is an ongoing process, and this guide should be updated as new security features are added or existing ones are modified.
