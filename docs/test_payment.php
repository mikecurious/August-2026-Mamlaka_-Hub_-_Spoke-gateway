<?php
/**
 * Payment API Test - Single File with UI
 * Combines HTML interface and PHP backend for testing payments
 */

// Handle AJAX requests
if ($_SERVER['REQUEST_METHOD'] === 'POST' && isset($_POST['action'])) {
    header('Content-Type: application/json');
    
    $action = $_POST['action'];
    $baseUrl = $_POST['baseUrl'] ?? 'https://payments.mam-laka.com/api/v1';
    $username = $_POST['username'] ?? 'app';
    $password = $_POST['password'] ?? 'cometappmain';
    
    if ($action === 'getToken') {
        $result = getToken($baseUrl, $username, $password);
        echo json_encode($result);
        exit;
    }
    
    if ($action === 'initiatePayment') {
        $token = $_POST['token'] ?? '';
        $paymentData = json_decode($_POST['paymentData'], true);
        $result = initiatePayment($baseUrl, $token, $paymentData);
        echo json_encode($result);
        exit;
    }
    
    if ($action === 'pollStatus') {
        $merchantId = $_POST['merchantId'] ?? '';
        $secureId = $_POST['secureId'] ?? '';
        $result = pollTransactionStatus($baseUrl, $merchantId, $secureId);
        echo json_encode($result);
        exit;
    }
}

// Helper functions
function makeRequest($url, $method = 'GET', $data = null, $headers = []) {
    $ch = curl_init();
    curl_setopt($ch, CURLOPT_URL, $url);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_HTTPHEADER, $headers);
    curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false);
    
    if ($method === 'POST') {
        curl_setopt($ch, CURLOPT_POST, true);
        if ($data !== null) {
            curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($data));
        }
    }
    
    $response = curl_exec($ch);
    $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
    $error = curl_error($ch);
    curl_close($ch);
    
    if ($error) {
        return ['error' => $error, 'http_code' => 0];
    }
    
    return [
        'http_code' => $httpCode,
        'body' => $response,
        'data' => json_decode($response, true)
    ];
}

function getToken($baseUrl, $username, $password) {
    // Login endpoint is at the base URL (router.GET("/", LoginHandler))
    // Ensure URL ends with / to avoid redirect issues
    $loginUrl = rtrim($baseUrl, '/') . '/';
    
    // Simple curl request with redirect following
    $ch = curl_init();
    curl_setopt($ch, CURLOPT_URL, $loginUrl);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
    curl_setopt($ch, CURLOPT_FOLLOWLOCATION, true); // Follow redirects
    curl_setopt($ch, CURLOPT_MAXREDIRS, 5);
    curl_setopt($ch, CURLOPT_SSL_VERIFYPEER, false);
    curl_setopt($ch, CURLOPT_HTTPHEADER, [
        'Authorization: Basic ' . base64_encode($username . ':' . $password),
        'Content-Type: application/json'
    ]);
    
    // Try GET first (as per route registration)
    curl_setopt($ch, CURLOPT_HTTPGET, true);
    $response = curl_exec($ch);
    $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
    $error = curl_error($ch);
    
    // If GET fails with 405, try POST
    if ($httpCode === 405) {
        curl_setopt($ch, CURLOPT_POST, true);
        curl_setopt($ch, CURLOPT_HTTPGET, false);
        $response = curl_exec($ch);
        $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
        $error = curl_error($ch);
    }
    
    curl_close($ch);
    
    if ($error) {
        return ['success' => false, 'error' => 'CURL Error: ' . $error, 'http_code' => 0];
    }
    
    $data = json_decode($response, true);
    
    if ($httpCode === 200 && isset($data['token'])) {
        return ['success' => true, 'token' => $data['token']];
    }
    
    // Return error details
    $error = 'Unknown error';
    if (isset($data['error'])) {
        $error = $data['error'];
        if (isset($data['details'])) {
            $error .= ': ' . $data['details'];
        }
    } else if ($httpCode > 0) {
        $error = 'HTTP ' . $httpCode . ': ' . (substr($response, 0, 200) ?? 'No response body');
    }
    
    return ['success' => false, 'error' => $error, 'http_code' => $httpCode, 'body' => $response];
}

function initiatePayment($baseUrl, $token, $paymentData) {
    $url = $baseUrl . '/pay';
    $headers = [
        'Authorization: Bearer ' . $token,
        'Content-Type: application/json'
    ];
    
    $response = makeRequest($url, 'POST', $paymentData, $headers);
    
    if ($response['http_code'] === 200 && isset($response['data']['secureId'])) {
        return [
            'success' => true,
            'data' => $response['data']
        ];
    }
    
    return [
        'success' => false,
        'error' => $response['data']['error'] ?? 'Failed to initiate payment',
        'http_code' => $response['http_code']
    ];
}

function pollTransactionStatus($baseUrl, $merchantId, $secureId, $maxAttempts = 30) {
    $url = $baseUrl . '/transaction?merchant=' . urlencode($merchantId) . '&secureId=' . urlencode($secureId);
    $headers = ['Content-Type: application/json'];
    
    for ($attempt = 1; $attempt <= $maxAttempts; $attempt++) {
        $response = makeRequest($url, 'GET', null, $headers);
        
        if ($response['http_code'] === 200 && isset($response['data'])) {
            // Handle nested transaction object
            $transactionData = $response['data'];
            $transaction = isset($transactionData['transaction']) ? $transactionData['transaction'] : $transactionData;
            
            // Map snake_case to camelCase and check status
            $status = '';
            if (isset($transaction['transaction_status'])) {
                // Map "success" to "COMPLETE", "failed" to "FAILED"
                $rawStatus = strtolower($transaction['transaction_status']);
                if ($rawStatus === 'success') {
                    $status = 'COMPLETE';
                } else if ($rawStatus === 'failed') {
                    $status = 'FAILED';
                } else {
                    $status = strtoupper($rawStatus);
                }
            } else if (isset($transaction['transactionStatus'])) {
                $status = $transaction['transactionStatus'];
            }
            
            // Normalize transaction data to expected format
            $normalizedTransaction = [
                'amount' => $transaction['amount'] ?? 0,
                'currency' => $transaction['currency'] ?? '',
                'externalId' => $transaction['external_id'] ?? $transaction['externalId'] ?? '',
                'netAmount' => $transaction['net_amount'] ?? $transaction['netAmount'] ?? ($transaction['amount'] ?? 0),
                'secureId' => $transaction['secure_id'] ?? $transaction['secureId'] ?? '',
                'transactionReport' => isset($transaction['transaction_report']) ? 
                    strtoupper($transaction['transaction_report']) : 
                    ($transaction['transactionReport'] ?? ''),
                'transactionStatus' => $status
            ];
            
            // Add reason if transaction failed
            if ($status === 'FAILED' && isset($transaction['reason'])) {
                $normalizedTransaction['reason'] = $transaction['reason'];
            }
            
            if ($status === 'COMPLETE' || $status === 'FAILED') {
                return [
                    'success' => true,
                    'complete' => true,
                    'data' => $normalizedTransaction,
                    'attempt' => $attempt
                ];
            }
        }
        
        // Stop after max attempts, even if status hasn't changed
        if ($attempt >= $maxAttempts) {
            // Return current status even if not COMPLETE/FAILED
            if (isset($transaction)) {
                return [
                    'success' => true,
                    'complete' => false,
                    'data' => $normalizedTransaction ?? [],
                    'attempt' => $attempt,
                    'error' => 'Maximum polling attempts reached (30). Current status: ' . ($status ?? 'unknown')
                ];
            }
            break;
        }
        
        sleep(3);
    }
    
    return [
        'success' => false,
        'complete' => false,
        'error' => 'Maximum polling attempts reached (30)'
    ];
}
?>
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Payment Test - Mam-laka</title>
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
            padding: 20px;
            display: flex;
            justify-content: center;
            align-items: center;
        }

        .container {
            background: white;
            border-radius: 20px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            max-width: 600px;
            width: 100%;
            padding: 40px;
            animation: slideUp 0.5s ease-out;
        }

        @keyframes slideUp {
            from {
                opacity: 0;
                transform: translateY(30px);
            }
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        h1 {
            color: #333;
            margin-bottom: 10px;
            font-size: 28px;
        }

        .subtitle {
            color: #666;
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

        .required {
            color: #e74c3c;
        }

        input, select {
            width: 100%;
            padding: 12px 15px;
            border: 2px solid #e0e0e0;
            border-radius: 8px;
            font-size: 14px;
            transition: all 0.3s;
            font-family: inherit;
        }

        input:focus, select:focus {
            outline: none;
            border-color: #667eea;
            box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
        }

        .form-row {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 15px;
        }

        .btn {
            width: 100%;
            padding: 14px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 8px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s;
            margin-top: 10px;
            position: relative;
        }

        .btn:hover:not(:disabled) {
            transform: translateY(-2px);
            box-shadow: 0 10px 20px rgba(102, 126, 234, 0.3);
        }

        .btn:active:not(:disabled) {
            transform: translateY(0);
        }

        .btn:disabled {
            background: #ccc;
            cursor: not-allowed;
            transform: none;
        }

        .status {
            margin-top: 20px;
            padding: 15px;
            border-radius: 8px;
            display: none;
        }

        .status.show {
            display: block;
            animation: fadeIn 0.3s;
        }

        @keyframes fadeIn {
            from { opacity: 0; }
            to { opacity: 1; }
        }

        .status.success {
            background: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
        }

        .status.error {
            background: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
            white-space: pre-wrap;
            word-wrap: break-word;
        }

        .status.info {
            background: #d1ecf1;
            color: #0c5460;
            border: 1px solid #bee5eb;
        }

        .status.loading {
            background: #fff3cd;
            color: #856404;
            border: 1px solid #ffeaa7;
        }

        .spinner-container {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 15px;
            padding: 20px;
            background: #f8f9fa;
            border-radius: 8px;
            margin-top: 20px;
        }

        .spinner {
            width: 40px;
            height: 40px;
            border: 4px solid #f3f3f3;
            border-top: 4px solid #667eea;
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }

        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }

        .spinner-large {
            width: 60px;
            height: 60px;
            border: 5px solid #f3f3f3;
            border-top: 5px solid #667eea;
            border-right: 5px solid #764ba2;
            border-radius: 50%;
            animation: spin 0.8s linear infinite;
        }

        .pulse {
            animation: pulse 1.5s ease-in-out infinite;
        }

        @keyframes pulse {
            0%, 100% { opacity: 1; transform: scale(1); }
            50% { opacity: 0.7; transform: scale(1.05); }
        }

        .result-box {
            margin-top: 20px;
            padding: 20px;
            background: #f8f9fa;
            border-radius: 8px;
            border-left: 4px solid #667eea;
            display: none;
        }

        .result-box.show {
            display: block;
        }

        .result-box h3 {
            margin-bottom: 15px;
            color: #333;
        }

        .result-item {
            display: flex;
            justify-content: space-between;
            padding: 8px 0;
            border-bottom: 1px solid #e0e0e0;
        }

        .result-item:last-child {
            border-bottom: none;
        }

        .result-label {
            font-weight: 600;
            color: #666;
        }

        .result-value {
            color: #333;
            word-break: break-all;
        }

        .polling-status {
            margin-top: 15px;
            padding: 20px;
            background: linear-gradient(135deg, #e7f3ff 0%, #f0f8ff 100%);
            border-radius: 8px;
            display: none;
            text-align: center;
        }

        .polling-status.show {
            display: block;
            animation: fadeIn 0.3s;
        }

        .polling-text {
            font-size: 16px;
            color: #333;
            margin-bottom: 15px;
            font-weight: 500;
        }

        .polling-attempt {
            font-size: 14px;
            color: #666;
            margin-top: 10px;
        }

        .progress-bar {
            width: 100%;
            height: 8px;
            background: #e0e0e0;
            border-radius: 4px;
            overflow: hidden;
            margin-top: 15px;
        }

        .progress-fill {
            height: 100%;
            background: linear-gradient(90deg, #667eea, #764ba2);
            width: 0%;
            transition: width 0.3s;
            animation: pulse 1.5s ease-in-out infinite;
        }

        .config-section {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 30px;
        }

        .config-section h3 {
            margin-bottom: 15px;
            color: #333;
            font-size: 16px;
        }

        .toggle-config {
            background: #6c757d;
            color: white;
            border: none;
            padding: 8px 15px;
            border-radius: 5px;
            cursor: pointer;
            font-size: 12px;
            margin-bottom: 15px;
        }

        .config-content {
            display: none;
        }

        .config-content.show {
            display: block;
        }

        .dots {
            display: inline-block;
        }

        .dots span {
            display: inline-block;
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background: #667eea;
            margin: 0 3px;
            animation: bounce 1.4s infinite ease-in-out both;
        }

        .dots span:nth-child(1) { animation-delay: -0.32s; }
        .dots span:nth-child(2) { animation-delay: -0.16s; }

        @keyframes bounce {
            0%, 80%, 100% {
                transform: scale(0);
                opacity: 0.5;
            }
            40% {
                transform: scale(1);
                opacity: 1;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>💳 Payment Test</h1>
        <p class="subtitle">Test the unified payment API endpoint</p>

        <div class="config-section">
            <button class="toggle-config" onclick="toggleConfig()">⚙️ Configuration</button>
            <div class="config-content" id="configContent">
                <div class="form-group">
                    <label>API Base URL</label>
                    <input type="text" id="baseUrl" value="https://payments.mam-laka.com/api/v1" placeholder="https://payments.mam-laka.com/api/v1">
                </div>
                <div class="form-group">
                    <label>Username</label>
                    <input type="text" id="username" value="app" placeholder="app">
                </div>
                <div class="form-group">
                    <label>Password</label>
                    <input type="password" id="password" value="cometappmain" placeholder="password">
                </div>
                <div class="form-group">
                    <label>Webhook URL</label>
                    <input type="text" id="webhookUrl" value="https://webhook.site/26e18488-fb4a-4da9-b5a4-ec5bf5976f9f" placeholder="https://webhook.site/...">
                </div>
            </div>
        </div>

        <form id="paymentForm">
            <div class="form-group">
                <label>Merchant ID <span class="required">*</span></label>
                <input type="text" id="impalaMerchantId" value="app" required>
            </div>

            <div class="form-row">
                <div class="form-group">
                    <label>Country <span class="required">*</span></label>
                    <select id="country" required>
                        <option value="KE">Kenya (KE)</option>
                        <option value="UG" selected>Uganda (UG)</option>
                        <option value="TZ">Tanzania (TZ)</option>
                        <option value="RW">Rwanda (RW)</option>
                        <option value="ZM">Zambia (ZM)</option>
                        <option value="BF">Burkina Faso (BF)</option>
                        <option value="SN">Senegal (SN)</option>
                        <option value="CI">Ivory Coast (CI)</option>
                        <option value="CM">Cameroon (CM)</option>
                        <option value="GH">Ghana (GH)</option>
                        <option value="NG">Nigeria (NG)</option>
                    </select>
                </div>
                <div class="form-group">
                    <label>Currency <span class="required">*</span></label>
                    <select id="currency" required>
                        <option value="KES">KES - Kenyan Shilling</option>
                        <option value="UGX" selected>UGX - Ugandan Shilling</option>
                        <option value="TZS">TZS - Tanzanian Shilling</option>
                        <option value="RWF">RWF - Rwandan Franc</option>
                        <option value="ZMW">ZMW - Zambian Kwacha</option>
                        <option value="XAF">XAF - Central African Franc</option>
                        <option value="NGN">NGN - Nigerian Naira</option>
                        <option value="GHS">GHS - Ghanaian Cedi</option>
                    </select>
                </div>
            </div>

            <div class="form-group">
                <label>Amount <span class="required">*</span></label>
                <input type="number" id="amount" value="100" min="1" required>
            </div>

            <div class="form-group">
                <label>Customer Name</label>
                <input type="text" id="customerName" value="John Doe" placeholder="John Doe">
            </div>

            <div class="form-group">
                <label>Customer Email <span class="required">*</span></label>
                <input type="email" id="customerEmail" value="john.doe@example.com" required>
            </div>

            <div class="form-group">
                <label>Phone Number <span class="required">*</span></label>
                <input type="text" id="payerPhone" value="256771850050" placeholder="256771850050" required>
                <small style="color: #666; font-size: 12px; margin-top: 5px; display: block;">
                    Format: Country code + number (e.g., 254771850050 for Kenya, 256771850050 for Uganda)
                </small>
            </div>

            <div class="form-group">
                <label>Description</label>
                <input type="text" id="description" value="Payment for services" placeholder="Payment description">
            </div>

            <div class="form-group">
                <label>External ID <span class="required">*</span></label>
                <input type="text" id="externalId" required>
            </div>

            <button type="submit" class="btn" id="submitBtn">
                🚀 Initiate Payment
            </button>
        </form>

        <div class="status" id="status"></div>
        <div class="result-box" id="resultBox"></div>
        <div class="polling-status" id="pollingStatus"></div>
    </div>

    <script>
        // Generate unique external ID on page load
        document.getElementById('externalId').value = 'ext_' + Date.now();

        function toggleConfig() {
            const config = document.getElementById('configContent');
            config.classList.toggle('show');
        }

        function showStatus(message, type) {
            const status = document.getElementById('status');
            status.textContent = message;
            status.className = 'status show ' + type;
        }

        function hideStatus() {
            document.getElementById('status').classList.remove('show');
        }

        function showResult(data) {
            const resultBox = document.getElementById('resultBox');
            resultBox.innerHTML = `
                <h3>📋 Payment Response</h3>
                <div class="result-item">
                    <span class="result-label">Message:</span>
                    <span class="result-value">${data.message || 'N/A'}</span>
                </div>
                <div class="result-item">
                    <span class="result-label">Secure ID:</span>
                    <span class="result-value">${data.secureId || 'N/A'}</span>
                </div>
                <div class="result-item">
                    <span class="result-label">Transaction ID:</span>
                    <span class="result-value">${data.transactionId || 'N/A'}</span>
                </div>
            `;
            resultBox.classList.add('show');
        }

        function showPollingStatus(attempt, maxAttempts) {
            const pollingStatus = document.getElementById('pollingStatus');
            const percentage = Math.min((attempt / maxAttempts) * 100, 100);
            pollingStatus.innerHTML = `
                <div class="spinner-large"></div>
                <div class="polling-text">Waiting for transaction status<span class="dots"><span></span><span></span><span></span></span></div>
                <div class="polling-attempt">Attempt ${attempt} of ${maxAttempts} (${Math.round(percentage)}%)</div>
                <div class="progress-bar">
                    <div class="progress-fill" style="width: ${percentage}%"></div>
                </div>
            `;
            pollingStatus.classList.add('show');
        }

        function hidePollingStatus() {
            document.getElementById('pollingStatus').classList.remove('show');
        }

        function showFinalResult(transaction) {
            const resultBox = document.getElementById('resultBox');
            resultBox.innerHTML = `
                <h3>📊 Final Transaction Status</h3>
                <div class="result-item">
                    <span class="result-label">Status:</span>
                    <span class="result-value" style="font-weight: 600; color: ${transaction.transactionStatus === 'COMPLETE' ? '#28a745' : '#dc3545'}">
                        ${transaction.transactionStatus}
                    </span>
                </div>
                <div class="result-item">
                    <span class="result-label">Report:</span>
                    <span class="result-value">${transaction.transactionReport}</span>
                </div>
                <div class="result-item">
                    <span class="result-label">Amount:</span>
                    <span class="result-value">${transaction.amount} ${transaction.currency}</span>
                </div>
                <div class="result-item">
                    <span class="result-label">Net Amount:</span>
                    <span class="result-value">${transaction.netAmount} ${transaction.currency}</span>
                </div>
                <div class="result-item">
                    <span class="result-label">Secure ID:</span>
                    <span class="result-value">${transaction.secureId}</span>
                </div>
                <div class="result-item">
                    <span class="result-label">External ID:</span>
                    <span class="result-value">${transaction.externalId}</span>
                </div>
                ${transaction.reason ? `
                <div class="result-item">
                    <span class="result-label">Reason:</span>
                    <span class="result-value" style="color: #dc3545;">${transaction.reason}</span>
                </div>
                ` : ''}
            `;
            resultBox.classList.add('show');
        }

        async function makeRequest(action, data) {
            const formData = new FormData();
            formData.append('action', action);
            
            for (const key in data) {
                if (typeof data[key] === 'object') {
                    formData.append(key, JSON.stringify(data[key]));
                } else {
                    formData.append(key, data[key]);
                }
            }

            const response = await fetch('', {
                method: 'POST',
                body: formData
            });

            return await response.json();
        }

        document.getElementById('paymentForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const baseUrl = document.getElementById('baseUrl').value;
            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;
            const webhookUrl = document.getElementById('webhookUrl').value;

            const paymentData = {
                impalaMerchantId: document.getElementById('impalaMerchantId').value,
                country: document.getElementById('country').value,
                currency: document.getElementById('currency').value,
                amount: parseInt(document.getElementById('amount').value),
                customerName: document.getElementById('customerName').value,
                customerEmail: document.getElementById('customerEmail').value,
                payerPhone: document.getElementById('payerPhone').value,
                description: document.getElementById('description').value,
                externalId: document.getElementById('externalId').value,
                callbackUrl: webhookUrl,
                redirectUrl: webhookUrl
            };

            const submitBtn = document.getElementById('submitBtn');
            submitBtn.disabled = true;
            submitBtn.textContent = '⏳ Processing...';

            try {
                // Step 1: Get token
                showStatus('🔐 Getting authentication token...', 'loading');
                const tokenResult = await makeRequest('getToken', {
                    baseUrl: baseUrl,
                    username: username,
                    password: password
                });

                if (!tokenResult.success) {
                    let errorMsg = tokenResult.error || 'Unknown error';
                    if (tokenResult.http_code) {
                        errorMsg += ' (HTTP ' + tokenResult.http_code + ')';
                    }
                    if (tokenResult.body) {
                        errorMsg += '\nResponse: ' + tokenResult.body.substring(0, 200);
                    }
                    showStatus('✗ Failed to get token: ' + errorMsg, 'error');
                    console.error('Token error details:', tokenResult);
                    submitBtn.disabled = false;
                    submitBtn.textContent = '🚀 Initiate Payment';
                    return;
                }

                const token = tokenResult.token;
                showStatus('✓ Token obtained successfully', 'success');

                // Step 2: Initiate payment
                showStatus('💳 Initiating payment...', 'loading');
                const paymentResult = await makeRequest('initiatePayment', {
                    baseUrl: baseUrl,
                    token: token,
                    paymentData: paymentData
                });

                if (!paymentResult.success) {
                    showStatus('✗ Failed to initiate payment: ' + (paymentResult.error || 'Unknown error'), 'error');
                    submitBtn.disabled = false;
                    submitBtn.textContent = '🚀 Initiate Payment';
                    return;
                }

                showStatus('✓ Payment initiated successfully', 'success');
                showResult(paymentResult.data);

                // Step 3: Poll transaction status
                showStatus('🔄 Polling transaction status...', 'info');
                
                let attempt = 0;
                const maxAttempts = 30;
                const pollInterval = setInterval(async () => {
                    attempt++;
                    showPollingStatus(attempt, maxAttempts);

                    const statusResult = await makeRequest('pollStatus', {
                        baseUrl: baseUrl,
                        merchantId: paymentData.impalaMerchantId,
                        secureId: paymentResult.data.secureId
                    });

                    if (statusResult.success && statusResult.complete) {
                        clearInterval(pollInterval);
                        hidePollingStatus();
                        
                        if (statusResult.data.transactionStatus === 'COMPLETE') {
                            showStatus('✅ Transaction completed successfully!', 'success');
                        } else {
                            showStatus('❌ Transaction failed!', 'error');
                        }
                        
                        showFinalResult(statusResult.data);
                        submitBtn.disabled = false;
                        submitBtn.textContent = '🚀 Initiate Payment';
                    } else if (attempt >= maxAttempts) {
                        clearInterval(pollInterval);
                        hidePollingStatus();
                        const errorMsg = statusResult.error || 'Maximum polling attempts reached (30)';
                        showStatus('⚠ ' + errorMsg, 'error');
                        if (statusResult.data) {
                            showFinalResult(statusResult.data);
                        }
                        submitBtn.disabled = false;
                        submitBtn.textContent = '🚀 Initiate Payment';
                    }
                }, 3000);

            } catch (error) {
                showStatus('✗ Error: ' + error.message, 'error');
                submitBtn.disabled = false;
                submitBtn.textContent = '🚀 Initiate Payment';
            }
        });
    </script>
</body>
</html>
