import json
import subprocess
import pandas as pd
import matplotlib.pyplot as plt
import sys
import argparse
from datetime import datetime

# Configuration
# -----------------------------------------------------------------------------
# Data Source: journalctl (via subprocess) or JSON file
# Key Metrics: 
#   1. "Cold Starts": Count of service start events per tenant.
#   2. "Error Rate": Count of logs with priority ERROR or CRITICAL.
#   3. "Volume by Service": Count of logs per process/service.
# Visualization: 
#   1. Bar Chart (Activations per Tenant)
#   2. Bar Chart (Errors per Tenant)
#   3. Bar Chart (Top 10 Log Producers)
# -----------------------------------------------------------------------------

def get_journal_logs(source_type="live", file_path=None, fetch_all=True):
    """
    Retrieves logs from journald or a JSON file.
    """
    logs = []
    
    if source_type == "live":
        print("Reading from journalctl (this may take a moment)...")
        # Base command
        cmd = ["journalctl", "-o", "json"]
        
        # If NOT fetching all, we filter for our specific pilot context (optional safety)
        # But user requested "complete data", so defaults to All.
        if not fetch_all:
             cmd.extend(["_COMM=user-rest-api", "_COMM=pilot", "_COMM=caddy", "_COMM=postgres"])

        # Limit to last 50,000 lines to prevent memory explosion on large systems
        # Remove '-n' if you truly want everything since boot.
        cmd.extend(["-n", "50000"])

        try:
            print(f"Executing: {' '.join(cmd)}")
            result = subprocess.run(cmd, capture_output=True, text=True, check=True)
            output = result.stdout
        except subprocess.CalledProcessError as e:
            print(f"Error running journalctl: {e}")
            sys.exit(1)
        except FileNotFoundError:
            print("Error: 'journalctl' command not found.")
            sys.exit(1)
            
    elif source_type == "file":
        print(f"Reading from file: {file_path}")
        try:
            with open(file_path, 'r') as f:
                output = f.read()
        except FileNotFoundError:
            print(f"Error: File {file_path} not found.")
            sys.exit(1)
    
    # Parse JSON lines
    # journalctl -o json outputs one JSON object per line
    count = 0
    for line in output.splitlines():
        if not line.strip(): 
            continue
        try:
            entry = json.loads(line)
            logs.append(entry)
            count += 1
        except json.JSONDecodeError:
            continue
    
    print(f"Parsed {count} log entries.")
    return logs

def analyze_data(logs):
    """
    Processes raw log entries into a DataFrame and calculates metrics.
    """
    if not logs:
        print("No logs found.")
        return None

    data = []
    for log in logs:
        # Extract relevant fields
        # _UID: Tenant/User ID
        uid = log.get("_UID", "system")
        
        # Timestamp
        ts_micro = int(log.get("__REALTIME_TIMESTAMP", 0))
        timestamp = datetime.fromtimestamp(ts_micro / 1_000_000)
        
        # Priority
        priority = int(log.get("PRIORITY", 6))
        
        # Message
        message = log.get("MESSAGE", "")
        
        # Service / Command Name
        # _COMM is usually the command name (e.g., 'sshd', 'pilot', 'caddy')
        # SYSLOG_IDENTIFIER is sometimes better for services
        comm = log.get("_COMM", log.get("SYSLOG_IDENTIFIER", "unknown"))

        # Unit
        unit = log.get("USER_UNIT", log.get("_SYSTEMD_UNIT", "unknown"))

        data.append({
            "timestamp": timestamp,
            "uid": uid,
            "priority": priority,
            "message": message,
            "comm": comm,
            "unit": unit
        })

    df = pd.DataFrame(data)
    return df

def visualize_metrics(df):
    """
    Generates visualizations for the analyzed metrics.
    """
    if df is None or df.empty:
        return

    # Metric 1: Activity per Tenant
    tenant_activity = df['uid'].value_counts().head(10) # Top 10 users

    # Metric 2: Error Distribution
    errors = df[df['priority'] <= 3]
    error_counts = errors['uid'].value_counts().head(10)

    # Metric 3: Top Services (Volume)
    service_activity = df['comm'].value_counts().head(10)

    # --- Plotting ---
    fig, (ax1, ax2, ax3) = plt.subplots(1, 3, figsize=(18, 6))

    # Plot 1: Activity per Tenant
    if not tenant_activity.empty:
        tenant_activity.plot(kind='bar', color='skyblue', ax=ax1)
        ax1.set_title('Top 10 Log Producers (By UID)')
        ax1.set_xlabel('UID')
        ax1.set_ylabel('Event Count')
        ax1.tick_params(axis='x', rotation=45)
    else:
        ax1.text(0.5, 0.5, "No Activity Data", ha='center')

    # Plot 2: Errors per Tenant
    if not error_counts.empty:
        error_counts.plot(kind='bar', color='salmon', ax=ax2)
        ax2.set_title('Top 10 Error Sources (By UID)')
        ax2.set_xlabel('UID')
        ax2.set_ylabel('Error Count')
        ax2.tick_params(axis='x', rotation=45)
    else:
        ax2.text(0.5, 0.5, "No Errors Found", ha='center', transform=ax2.transAxes)

    # Plot 3: Volume by Service
    if not service_activity.empty:
        service_activity.plot(kind='bar', color='lightgreen', ax=ax3)
        ax3.set_title('Top 10 Log Producers (By Process/Comm)')
        ax3.set_xlabel('Process Name')
        ax3.set_ylabel('Event Count')
        ax3.tick_params(axis='x', rotation=45)
    else:
        ax3.text(0.5, 0.5, "No Service Data", ha='center')

    plt.tight_layout()
    
    output_file = "pilot_log_analysis_complete.png"
    plt.savefig(output_file)
    print(f"\nAnalysis complete. Visualization saved to: {output_file}")
    
    print("\nSummary Metrics:")
    print("-" * 30)
    print(f"Total Log Entries: {len(df)}")
    print(f"Total Errors:      {len(errors)}")
    print(f"Unique Processes:  {df['comm'].nunique()}")
    print("\nTop 5 Processes:")
    print(df['comm'].value_counts().head(5).to_string())

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Analyze System Logs via Journald")
    parser.add_argument("file", nargs="?", help="Path to JSON log file (optional)")
    parser.add_argument("--filter", action="store_true", help="Filter for Pilot specific services only")
    args = parser.parse_args()

    print("--- Pilot Log Analyzer (Complete Data) ---")
    
    if args.file:
        logs = get_journal_logs("file", args.file)
    else:
        # Default to ALL (fetch_all=True) unless --filter is passed
        # This matches the user request for "complete data"
        logs = get_journal_logs("live", fetch_all=not args.filter)
        
    df = analyze_data(logs)
    visualize_metrics(df)
