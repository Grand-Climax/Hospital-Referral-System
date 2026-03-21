package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const landingPageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Hospital Referral System API</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;600;800&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-color: #0b0f19;
            --text-primary: #f8fafc;
            --text-secondary: #94a3b8;
            --accent-color: #3b82f6;
            --accent-hover: #2563eb;
            --card-bg: rgba(30, 41, 59, 0.7);
            --border-color: rgba(255, 255, 255, 0.1);
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: 'Inter', sans-serif;
            background-color: var(--bg-color);
            color: var(--text-primary);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            background-image: 
                radial-gradient(circle at 15% 50%, rgba(59, 130, 246, 0.15), transparent 25%),
                radial-gradient(circle at 85% 30%, rgba(147, 51, 234, 0.15), transparent 25%);
            background-attachment: fixed;
            line-height: 1.6;
            overflow: hidden;
        }

        .container {
            max-width: 800px;
            padding: 4rem;
            background: var(--card-bg);
            border: 1px solid var(--border-color);
            border-radius: 24px;
            backdrop-filter: blur(20px);
            -webkit-backdrop-filter: blur(20px);
            box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
            text-align: center;
            animation: fadeUp 0.8s ease-out forwards;
            opacity: 0;
            transform: translateY(20px);
        }

        @keyframes fadeUp {
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        .status-badge {
            display: inline-flex;
            align-items: center;
            padding: 0.5rem 1rem;
            background: rgba(16, 185, 129, 0.1);
            color: #10b981;
            border: 1px solid rgba(16, 185, 129, 0.2);
            border-radius: 9999px;
            font-size: 0.875rem;
            font-weight: 600;
            margin-bottom: 2rem;
            letter-spacing: 0.05em;
            text-transform: uppercase;
        }

        .status-dot {
            width: 8px;
            height: 8px;
            background-color: #10b981;
            border-radius: 50%;
            margin-right: 8px;
            box-shadow: 0 0 10px #10b981;
            animation: pulse 2s infinite;
        }

        @keyframes pulse {
            0% { opacity: 1; transform: scale(1); }
            50% { opacity: 0.5; transform: scale(1.2); }
            100% { opacity: 1; transform: scale(1); }
        }

        h1 {
            font-size: 3rem;
            font-weight: 800;
            letter-spacing: -0.025em;
            margin-bottom: 1rem;
            background: linear-gradient(135deg, #ffffff 0%, #94a3b8 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }

        p {
            color: var(--text-secondary);
            font-size: 1.125rem;
            max-width: 600px;
            margin: 0 auto 3rem auto;
            font-weight: 300;
        }

        .button-group {
            display: flex;
            gap: 1rem;
            justify-content: center;
            flex-wrap: wrap;
        }

        .btn {
            display: inline-flex;
            align-items: center;
            justify-content: center;
            padding: 0.875rem 2rem;
            font-size: 1rem;
            font-weight: 600;
            border-radius: 12px;
            text-decoration: none;
            transition: all 0.2s ease;
        }

        .btn-primary {
            background-color: var(--accent-color);
            color: white;
            border: 1px solid transparent;
            box-shadow: 0 4px 14px 0 rgba(59, 130, 246, 0.39);
        }

        .btn-primary:hover {
            background-color: var(--accent-hover);
            transform: translateY(-2px);
            box-shadow: 0 6px 20px rgba(59, 130, 246, 0.4);
        }

        .btn-secondary {
            background-color: transparent;
            color: var(--text-primary);
            border: 1px solid var(--border-color);
        }

        .btn-secondary:hover {
            background-color: rgba(255, 255, 255, 0.05);
            border-color: rgba(255, 255, 255, 0.2);
            transform: translateY(-2px);
        }

        .footer {
            margin-top: 4rem;
            font-size: 0.875rem;
            color: rgba(148, 163, 184, 0.6);
            display: flex;
            justify-content: center;
            gap: 2rem;
        }

        .footer span {
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        @media (max-width: 640px) {
            .container {
                padding: 2.5rem 1.5rem;
                margin: 1rem;
            }
            h1 { font-size: 2.25rem; }
            .button-group { flex-direction: column; width: 100%; }
            .btn { width: 100%; }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="status-badge">
            <div class="status-dot"></div>
            System Operational
        </div>
        
        <h1>Hospital Referral API</h1>
        <p>A high-performance enterprise backend architecture powering patient transfers, network routing, and institutional coordination securely.</p>
        
        <div class="button-group">
            <a href="/swagger/index.html" class="btn btn-primary">
                Explore API Documentation
            </a>
            <a href="/health" class="btn btn-secondary">
                View Health Metrics
            </a>
        </div>
        
        <div class="footer">
            <span>v1.0.0</span>
            <span>REST API</span>
            <span>Golang Server</span>
        </div>
    </div>
</body>
</html>`

// HomeHandler serves the visually stunning, modern HTML landing page.
func HomeHandler(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(landingPageHTML))
}
