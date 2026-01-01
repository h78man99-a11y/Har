<?php
// --- CONFIGURATION ---
$botToken    = "7960235034:AAGspuayD8vd-CnAkGp1LjpUv2RhcoopqKU"; // Generate a NEW token for safety!
$chatID      = "-1002303170594";
$securityKey = "MySecretKey123";
$logFile     = "conversions.txt";

// --- SECURITY CHECK ---
if (($_GET['key'] ?? '') !== $securityKey) {
    header('HTTP/1.1 403 Forbidden');
    exit("Unauthorized Access.");
}

// --- DATA CAPTURE ---
$user_upi    = $_GET['upi']    ?? 'N/A';
$refer_upi   = $_GET['refer']  ?? 'N/A';
$offer_name  = $_GET['offer']  ?? 'Unknown Offer';
$panel_ip    = $_GET['user_ip']   ?? $_SERVER['REMOTE_ADDR'];
$comp_time   = date("d-M-Y h:i:s A");

// --- SAVE TO FILE ---
$fileData = "Time: $comp_time | Offer: $offer_name | User: $user_upi | Refer: $refer_upi | IP: $panel_ip" . PHP_EOL;
file_put_contents($logFile, $fileData, FILE_APPEND | LOCK_EX);

// --- TELEGRAM NOTIFICATION ---
$message  = "✅ *NEW CONVERSION SAVED* ✅\n\n";
$message .= "🏢 *Offer:* " . strtoupper($offer_name) . "\n";
$message .= "👤 *User UPI:* `{$user_upi}`\n";
$message .= "🔗 *Refer:* `{$refer_upi}`\n";
$message .= "🕒 *Time:* {$comp_time}\n\n";
$message .= "📂 _Saved to database_";

$url = "https://api.telegram.org/bot$botToken/sendMessage";
$data = ['chat_id' => $chatID, 'text' => $message, 'parse_mode' => 'Markdown'];

$ch = curl_init($url);
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
curl_setopt($ch, CURLOPT_POSTFIELDS, $data);
curl_exec($ch);
curl_close($ch);

echo "OK";
?>
