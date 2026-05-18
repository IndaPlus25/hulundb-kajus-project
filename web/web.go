package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

type ResolveRequest struct {
	Domain     string `json:"domain"`
	RecordType string `json:"recordType"`
}

type ResolveResponse struct {
	Domain  string      `json:"domain"`
	Results interface{} `json:"results"`
	Error   string      `json:"error,omitempty"`
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

	if req.RecordType == "" {
		req.RecordType = "A"
	}

	log.Printf("Resolving %s (%s)", req.Domain, req.RecordType)

	var results []string
	var err error

	switch req.RecordType {
	case "A":
		results, err = resolveA(req.Domain)
	case "AAAA":
		results, err = resolveAAAA(req.Domain)
	case "MX":
		mxResults, err := resolveMX(req.Domain)
		if err != nil {
			resp := ResolveResponse{
				Domain: req.Domain,
				Error:  fmt.Sprintf("MX lookup failed: %v", err),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ResolveResponse{
			Domain:  req.Domain,
			Results: mxResults,
		})
		return
	case "NS":
		results, err = resolveNS(req.Domain)
	case "CNAME":
		results, err = resolveCNAME(req.Domain)
	default:
		results, err = resolveA(req.Domain)
	}

	if err != nil {
		resp := ResolveResponse{
			Domain: req.Domain,
			Error:  fmt.Sprintf("Lookup failed: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	if len(results) == 0 {
		resp := ResolveResponse{
			Domain: req.Domain,
			Error:  "No records found",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	log.Printf("Found %d results for %s", len(results), req.Domain)
	resp := ResolveResponse{
		Domain:  req.Domain,
		Results: results,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func resolveA(domain string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, domain)
	if err != nil {
		return nil, err
	}

	var results []string
	for _, ip := range ips {
		if ip.IP.To4() != nil {
			results = append(results, ip.IP.String())
		}
	}
	return results, nil
}

func resolveAAAA(domain string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, domain)
	if err != nil {
		return nil, err
	}

	var results []string
	for _, ip := range ips {
		if ip.IP.To4() == nil && ip.IP.To16() != nil {
			results = append(results, ip.IP.String())
		}
	}
	return results, nil
}

func resolveMX(domain string) ([]map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mxs, err := net.DefaultResolver.LookupMX(ctx, domain)
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for _, mx := range mxs {
		results = append(results, map[string]interface{}{
			"preference": mx.Pref,
			"exchange":   mx.Host,
		})
	}
	return results, nil
}

func resolveNS(domain string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	nss, err := net.DefaultResolver.LookupNS(ctx, domain)
	if err != nil {
		return nil, err
	}

	var results []string
	for _, ns := range nss {
		results = append(results, ns.Host)
	}
	return results, nil
}

func resolveCNAME(domain string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cname, err := net.DefaultResolver.LookupCNAME(ctx, domain)
	if err != nil {
		return nil, err
	}

	return []string{cname}, nil
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

		select {
			width: 100%;
			padding: 12px 16px;
			border: 2px solid #e0e0e0;
			border-radius: 8px;
			font-size: 16px;
			transition: border-color 0.3s;
			background-color: white;
			cursor: pointer;
		}

		select:focus {
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
		}

		.ip-address:last-child {
			margin-bottom: 0;
		}

		.record-item {
			font-family: 'Monaco', 'Menlo', monospace;
			font-size: 14px;
			padding: 10px;
			background: white;
			border-radius: 6px;
			margin-bottom: 8px;
			word-break: break-all;
		}

		.record-item:last-child {
			margin-bottom: 0;
		}

		.record-item strong {
			color: #667eea;
			margin-right: 8px;
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
			padding: 20px;
		}

		.loading.show {
			display: block;
		}

		.spinner {
			display: inline-block;
			width: 32px;
			height: 32px;
			border: 4px solid #e0e0e0;
			border-top-color: #667eea;
			border-radius: 50%;
			animation: spin 0.8s linear infinite;
			margin-bottom: 12px;
		}

		@keyframes spin {
			to { transform: rotate(360deg); }
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>DNS Resolver</h1>
		<p class="subtitle">DNS lookup tool</p>

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
			<div class="form-group">
				<label for="recordType">Record-typ</label>
				<select id="recordType">
					<option value="A">A (IPv4)</option>
					<option value="AAAA">AAAA (IPv6)</option>
					<option value="MX">MX (Mail)</option>
					<option value="NS">NS (Nameserver)</option>
					<option value="CNAME">CNAME (Alias)</option>
				</select>
			</div>
			<button type="submit" id="submitBtn">Slå upp</button>
		</form>

		<div class="loading" id="loading">
			<div class="spinner"></div>
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
		const recordTypeSelect = document.getElementById('recordType');
		const submitBtn = document.getElementById('submitBtn');
		const loading = document.getElementById('loading');
		const result = document.getElementById('result');
		const resultDomain = document.getElementById('resultDomain');
		const resultContent = document.getElementById('resultContent');

		form.addEventListener('submit', async (e) => {
			e.preventDefault();

			const domain = domainInput.value.trim();
			const recordType = recordTypeSelect.value;
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
					body: JSON.stringify({ domain, recordType }),
				});

				const data = await response.json();
				resultDomain.textContent = domain + ' (' + recordType + ')';

				if (data.error) {
					result.classList.add('error');
					resultContent.innerHTML = '<div class="error-message">' + data.error + '</div>';
				} else if (data.results && Array.isArray(data.results) && data.results.length > 0) {
					result.classList.add('success');
					if (recordType === 'MX') {
						resultContent.innerHTML = data.results
							.map(mx => '<div class="record-item"><strong>' + mx.preference + '</strong> - ' + mx.exchange + '</div>')
							.join('');
					} else {
						resultContent.innerHTML = data.results
							.map(item => '<div class="ip-address">' + item + '</div>')
							.join('');
					}
				} else {
					result.classList.add('error');
					resultContent.innerHTML = '<div class="error-message">Ingen record hittad</div>';
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

		domainInput.focus();
	</script>
</body>
</html>
`
