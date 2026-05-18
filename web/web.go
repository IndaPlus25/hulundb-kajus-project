package web

import (
	"encoding/json"
	"fmt"
	"hulundb-kajus-dns/dns"
	"hulundb-kajus-dns/resolver"
	"net/http"
)

type ResolveRequest struct {
	Domain string `json:"domain"`
}

type ResolveResponse struct {
	Domain string   `json:"domain"`
	IPs    []string `json:"ips"`
	Error  string   `json:"error,omitempty"`
}

func Start(addr string) error {
	http.HandleFunc("/", serveHTML)
	http.HandleFunc("/api/resolve", handleResolve)

	fmt.Println("Web server listening on", addr)
	return http.ListenAndServe(addr, nil)
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, htmlContent)
}

func handleResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ResolveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Domain == "" {
		http.Error(w, "Domain is required", http.StatusBadRequest)
		return
	}

	query, err := dns.BuildQuery(req.Domain, dns.TypeA)
	if err != nil {
		resp := ResolveResponse{
			Domain: req.Domain,
			Error:  fmt.Sprintf("Failed to build query: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	response, err := resolver.Resolve(query, 0)
	if err != nil {
		resp := ResolveResponse{
			Domain: req.Domain,
			Error:  fmt.Sprintf("Resolution failed: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	msg, err := dns.DecodeMessage(response)
	if err != nil {
		resp := ResolveResponse{
			Domain: req.Domain,
			Error:  fmt.Sprintf("Failed to decode response: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	var ips []string
	for _, answer := range msg.Answers {
		if aRecord, ok := answer.Data.(dns.ARecord); ok {
			ips = append(ips, aRecord.IP.String())
		}
	}

	resp := ResolveResponse{
		Domain: req.Domain,
		IPs:    ips,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

var htmlContent = `<!DOCTYPE html>
<html lang="sv">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>DNS Resolver</title>
	<style>
		* {
			margin: 0;
			padding: 0;
			box-sizing: border-box;
		}

		body {
			font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
			background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
			min-height: 100vh;
			display: flex;
			justify-content: center;
			align-items: center;
			padding: 20px;
		}

		.container {
			background: white;
			border-radius: 16px;
			box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
			padding: 40px;
			max-width: 500px;
			width: 100%;
		}

		h1 {
			text-align: center;
			color: #333;
			margin-bottom: 10px;
			font-size: 28px;
		}

		.subtitle {
			text-align: center;
			color: #888;
			margin-bottom: 30px;
			font-size: 14px;
		}

		.form-group {
			margin-bottom: 20px;
		}

		label {
			display: block;
			margin-bottom: 8px;
			color: #333;
			font-weight: 500;
			font-size: 14px;
		}

		input[type="text"] {
			width: 100%;
			padding: 12px 16px;
			border: 2px solid #e0e0e0;
			border-radius: 8px;
			font-size: 16px;
			transition: border-color 0.3s;
		}

		input[type="text"]:focus {
			outline: none;
			border-color: #667eea;
		}

		button {
			width: 100%;
			padding: 12px 16px;
			background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
			color: white;
			border: none;
			border-radius: 8px;
			font-size: 16px;
			font-weight: 600;
			cursor: pointer;
			transition: transform 0.2s, box-shadow 0.2s;
		}

		button:hover {
			transform: translateY(-2px);
			box-shadow: 0 10px 20px rgba(102, 126, 234, 0.3);
		}

		button:active {
			transform: translateY(0);
		}

		button:disabled {
			opacity: 0.6;
			cursor: not-allowed;
			transform: none;
		}

		.result {
			margin-top: 30px;
			padding: 20px;
			background: #f8f9fa;
			border-radius: 8px;
			display: none;
		}

		.result.show {
			display: block;
		}

		.result h2 {
			font-size: 14px;
			color: #666;
			margin-bottom: 12px;
			text-transform: uppercase;
			font-weight: 600;
		}

		.result.success {
			border-left: 4px solid #4caf50;
		}

		.result.error {
			border-left: 4px solid #f44336;
		}

		.ip-address {
			font-family: 'Monaco', 'Menlo', monospace;
			font-size: 18px;
			padding: 10px;
			background: white;
			border-radius: 6px;
			margin-bottom: 8px;
			word-break: break-all;
			cursor: pointer;
			transition: background 0.2s;
		}

		.ip-address:hover {
			background: #e8f5e9;
		}

		.ip-address:last-child {
			margin-bottom: 0;
		}

		.error-message {
			color: #f44336;
			font-size: 14px;
			padding: 10px;
			background: white;
			border-radius: 6px;
			word-break: break-all;
		}

		.loading {
			display: none;
			text-align: center;
			color: #667eea;
			font-size: 14px;
		}

		.loading.show {
			display: block;
		}

		.spinner {
			display: inline-block;
			width: 12px;
			height: 12px;
			border: 2px solid #e0e0e0;
			border-top-color: #667eea;
			border-radius: 50%;
			animation: spin 0.6s linear infinite;
			margin-right: 8px;
		}

		@keyframes spin {
			to { transform: rotate(360deg); }
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>DNS Resolver</h1>
		<p class="subtitle">Slå upp domäner med vår DNS-server</p>

		<form id="resolveForm">
			<div class="form-group">
				<label for="domain">Domännamn</label>
				<input
					type="text"
					id="domain"
					placeholder="e.g., google.com, github.com"
					required
				>
			</div>
			<button type="submit" id="submitBtn">Slå upp</button>
		</form>

		<div class="loading" id="loading">
			<span class="spinner"></span>
			Söker...
		</div>

		<div class="result" id="result">
			<h2 id="resultDomain"></h2>
			<div id="resultContent"></div>
		</div>
	</div>

	<script>
		const form = document.getElementById('resolveForm');
		const domainInput = document.getElementById('domain');
		const submitBtn = document.getElementById('submitBtn');
		const loading = document.getElementById('loading');
		const result = document.getElementById('result');
		const resultDomain = document.getElementById('resultDomain');
		const resultContent = document.getElementById('resultContent');

		form.addEventListener('submit', async (e) => {
			e.preventDefault();

			const domain = domainInput.value.trim();
			if (!domain) return;

			submitBtn.disabled = true;
			loading.classList.add('show');
			result.classList.remove('show', 'success', 'error');

			try {
				const response = await fetch('/api/resolve', {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json',
					},
					body: JSON.stringify({ domain }),
				});

				const data = await response.json();
				resultDomain.textContent = data.domain;

				if (data.error) {
					result.classList.add('error');
					resultContent.innerHTML = '<div class="error-message">' + data.error + '</div>';
				} else if (data.ips && data.ips.length > 0) {
					result.classList.add('success');
					resultContent.innerHTML = data.ips
						.map(ip => '<div class="ip-address">' + ip + '</div>')
						.join('');
				} else {
					result.classList.add('error');
					resultContent.innerHTML = '<div class="error-message">Ingen IP-adress hittad</div>';
				}

				result.classList.add('show');
			} catch (error) {
				result.classList.add('error');
				resultDomain.textContent = domain;
				resultContent.innerHTML = '<div class="error-message">Fel vid anslutning: ' + error.message + '</div>';
				result.classList.add('show');
			} finally {
				submitBtn.disabled = false;
				loading.classList.remove('show');
			}
		});

		// Focus on domain input on page load
		domainInput.focus();
	</script>
</body>
</html>
`
