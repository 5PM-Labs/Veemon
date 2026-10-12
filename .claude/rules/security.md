# Security Guidelines (OWASP Top 10)

This project follows OWASP Top 10 security guidelines. Reference: https://owasp.org/www-project-top-ten/

## A01:2021 - Broken Access Control

```go
// ✓ Good: Role-based access control middleware
func RequireRoles(roles ...string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        authCtx, ok := middleware.GetAuthContext(c)
        if !ok {
            return errors.Unauthorized("authentication required")
        }
        for _, role := range roles {
            if authCtx.HasRole(role) {
                return c.Next()
            }
        }
        return errors.Forbidden("insufficient permissions")
    }
}

// ✓ Good: Verify resource ownership
func (uc *UserUseCase) GetUserProfile(ctx context.Context, targetUserID string) (*entity.User, error) {
    // Users can only access their own profile unless they are admins.
    authUser, _ := middleware.AuthFromContext(ctx)
    if authUser == nil {
        return nil, errors.Unauthorized("authentication required")
    }
    if authUser.UserID != targetUserID && !authUser.HasRole("admin") {
        return nil, errors.Forbidden("cannot access other user's profile")
    }
    return uc.repo.FindByID(ctx, targetUserID)
}
```

## A02:2021 - Cryptographic Failures

```go
// ✓ Good: Use bcrypt for passwords
hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

// ✓ Good: Secure password comparison (constant-time)
err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))

// ✓ Good: Use strong JWT secret (min 256 bits)
// In .env: JWT_SECRET=<random-32-byte-hex-string>

// ✗ Bad: Weak or hardcoded secrets
const jwtSecret = "secret123"  // NEVER DO THIS
```

## A03:2021 - Injection

```go
// ✓ Good: Use parameterized queries (GORM)
db.Where("email = ?", email).First(&user)
db.Where("id IN ?", ids).Find(&users)

// ✗ Bad: String concatenation (SQL injection)
db.Where("email = '" + email + "'").First(&user)
db.Raw("SELECT * FROM users WHERE email = '" + email + "'")

// ✓ Good: Command injection prevention
// Avoid os/exec with user input. If necessary, whitelist allowed values
allowedCommands := map[string]bool{"status": true, "version": true}
if !allowedCommands[userInput] {
    return errors.BadRequest("invalid command")
}
```

## A04:2021 - Insecure Design

```go
// ✓ Good: Rate limiting to prevent abuse
app.Use(middleware.RateLimitMiddleware(middleware.RateLimitConfig{
    Max:      100,
    Duration: 1 * time.Minute,
}))

// ✓ Good: Implement account lockout
type LoginAttempt struct {
    FailedAttempts int
    LockedUntil    time.Time
}

// ✓ Good: Use secure defaults
cfg := Config{
    JWTExpiration: 24 * time.Hour,  // Not indefinite
    MaxLoginAttempts: 5,
    LockoutDuration: 15 * time.Minute,
}
```

## A05:2021 - Security Misconfiguration

```go
// ✓ Good: Disable debug in production
if cfg.Environment == "production" {
    gin.SetMode(gin.ReleaseMode)
    app.Settings.DisableStartupMessage = true
}

// ✓ Good: Secure CORS configuration
app.Use(cors.New(cors.Config{
    AllowOrigins:     cfg.CORSOrigins,  // Not "*" in production
    AllowMethods:     "GET,POST,PUT,DELETE",
    AllowHeaders:     "Origin,Content-Type,Authorization",
    AllowCredentials: true,
}))

// ✓ Good: Security headers
app.Use(helmet.New())  // Sets X-Content-Type-Options, X-Frame-Options, etc.

// ✓ Good: File permissions (0600 for sensitive files)
os.WriteFile(path, data, 0600)  // Not 0644 or 0777
```

## A06:2021 - Vulnerable and Outdated Components

```bash
# ✓ Good: Regularly update dependencies
go get -u ./...
go mod tidy

# ✓ Good: Check for vulnerabilities
govulncheck ./...

# ✓ Good: Use dependabot or renovate for automated updates
```

## A07:2021 - Identification and Authentication Failures

```go
// ✓ Good: Strong password validation
type RegisterRequest struct {
    Password string `validate:"required,min=8,password"`  // Custom validator
}

// Custom password validator: requires upper, lower, digit
func passwordValidator(fl validator.FieldLevel) bool {
    password := fl.Field().String()
    var hasUpper, hasLower, hasDigit bool
    for _, r := range password {
        switch {
        case r >= 'A' && r <= 'Z': hasUpper = true
        case r >= 'a' && r <= 'z': hasLower = true
        case r >= '0' && r <= '9': hasDigit = true
        }
    }
    return hasUpper && hasLower && hasDigit
}

// ✓ Good: mint tokens via pkg/token (PASETO v4 local, revocable jti, bounded expiry).
// The token service enforces a strong secret at construction; JWT_SECRET has no
// default and Config.Validate() fails fast on a weak/placeholder value.
tokenService, err := token.NewTokenService(cfg.JWTSecret, cfg.JWTExpiration)
if err != nil {
    return fmt.Errorf("init token service: %w", err)
}
accessToken, err := tokenService.GenerateToken(user.ID, user.Email, user.Roles, user.CompanyCode)

// Logout/refresh revoke the token by its jti via pkg/authguard (Redis-backed).
```

> Note: this project uses PASETO (`pkg/token`), not JWT. `golang-jwt` was removed;
> `JWT_SECRET` is only a (legacy) variable name for the PASETO signing key.

## A08:2021 - Software and Data Integrity Failures

```go
// ✓ Good: Validate input data integrity
func (uc *UseCase) ProcessPayment(ctx context.Context, req PaymentRequest) error {
    // Validate request signature if from external source
    if !verifySignature(req.Data, req.Signature, publicKey) {
        return errors.BadRequest("invalid signature")
    }
    // Process payment...
}

// ✓ Good: Use checksums for file uploads
func validateUpload(file multipart.File, expectedHash string) error {
    hash := sha256.New()
    if _, err := io.Copy(hash, file); err != nil {
        return err
    }
    if hex.EncodeToString(hash.Sum(nil)) != expectedHash {
        return errors.BadRequest("file integrity check failed")
    }
    return nil
}
```

## A09:2021 - Security Logging and Monitoring Failures

```go
// ✓ Good: Log security events
// Identify by user_id / a keyed email hash — never the raw email (PII, see data-classification.md)
logger.Warn("failed login attempt",
    zap.String("email_hash", hashForLog(email)),
    zap.String("ip", c.IP()),
    zap.String("user_agent", c.Get("User-Agent")),
)

logger.Info("user logged in",
    zap.String("user_id", user.ID),
    zap.String("ip", c.IP()),
)

// ✓ Good: Log access to sensitive data
logger.Info("user record accessed",
    zap.String("accessor_id", authCtx.UserID),
    zap.String("target_user_id", targetUserID),
    zap.String("action", "view_user"),
)

// ✗ Bad: Logging sensitive data
logger.Info("user login", zap.String("password", password))  // NEVER DO THIS
```

## A10:2021 - Server-Side Request Forgery (SSRF)

```go
// ✓ Good: Whitelist allowed URLs/hosts
var allowedHosts = map[string]bool{
    "api.example.com": true,
    "cdn.example.com": true,
}

func fetchURL(urlStr string) error {
    u, err := url.Parse(urlStr)
    if err != nil {
        return errors.BadRequest("invalid URL")
    }
    if !allowedHosts[u.Host] {
        return errors.Forbidden("host not allowed")
    }
    // Proceed with request...
}

// ✓ Good: Block internal IPs
func isInternalIP(ip net.IP) bool {
    privateBlocks := []string{
        "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.0/8",
    }
    for _, block := range privateBlocks {
        _, cidr, _ := net.ParseCIDR(block)
        if cidr.Contains(ip) {
            return true
        }
    }
    return false
}
```
