package report

const ExecutiveReportTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Executive Forensic Investigation Report - {{.CaseName}}</title>
    <style>
        @media print {
            body { background: #fff !important; color: #000 !important; }
            .no-print { display: none !important; }
            .page-break { page-break-before: always; }
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            line-height: 1.5;
            color: #1a1a1a;
            background-color: #f4f6f9;
            margin: 0;
            padding: 40px;
        }
        .container {
            max-width: 1000px;
            margin: 0 auto;
            background: #ffffff;
            padding: 50px;
            border-radius: 8px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.08);
        }
        .header {
            border-bottom: 3px solid #0078D4;
            padding-bottom: 20px;
            margin-bottom: 30px;
        }
        .header h1 {
            margin: 0;
            color: #0078D4;
            font-size: 28px;
        }
        .meta-table {
            width: 100%;
            margin-top: 15px;
            border-collapse: collapse;
        }
        .meta-table td {
            padding: 6px 0;
            font-size: 14px;
        }
        .meta-label {
            font-weight: bold;
            color: #555;
            width: 150px;
        }
        .section-title {
            font-size: 20px;
            color: #0078D4;
            border-bottom: 1px solid #e0e0e0;
            padding-bottom: 8px;
            margin-top: 40px;
            margin-bottom: 15px;
        }
        .summary-cards {
            display: flex;
            gap: 20px;
            margin-bottom: 30px;
        }
        .card {
            flex: 1;
            padding: 20px;
            background: #f8f9fa;
            border-radius: 6px;
            border-left: 4px solid #0078D4;
        }
        .card.critical { border-left-color: #C42B1C; }
        .card.warning { border-left-color: #9D5D00; }
        .card.success { border-left-color: #0F7B0F; }
        .card-num { font-size: 28px; font-weight: bold; margin-bottom: 5px; }
        .card-label { font-size: 13px; color: #666; text-transform: uppercase; letter-spacing: 0.5px; }
        table.data-table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 10px;
            font-size: 13px;
        }
        table.data-table th {
            background-color: #f1f3f5;
            text-align: left;
            padding: 10px;
            border: 1px solid #dee2e6;
            font-weight: 600;
        }
        table.data-table td {
            padding: 8px 10px;
            border: 1px solid #dee2e6;
            word-break: break-word;
        }
        .badge {
            display: inline-block;
            padding: 3px 8px;
            border-radius: 3px;
            font-size: 11px;
            font-weight: bold;
            color: #fff;
        }
        .badge-critical { background-color: #C42B1C; }
        .badge-high { background-color: #D83B01; }
        .badge-medium { background-color: #9D5D00; }
        .badge-low { background-color: #0078D4; }
        .footer {
            margin-top: 50px;
            border-top: 1px solid #eee;
            padding-top: 15px;
            font-size: 12px;
            color: #888;
            text-align: center;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Meteor Forensic Workbench</h1>
            <div style="font-size: 16px; color: #666; margin-top: 5px;">Executive Forensic Incident Report</div>
            <table class="meta-table">
                <tr><td class="meta-label">Case Name:</td><td>{{.CaseName}}</td></tr>
                <tr><td class="meta-label">Investigator:</td><td>{{.Investigator}}</td></tr>
                <tr><td class="meta-label">Generated On:</td><td>{{.GeneratedAt}}</td></tr>
            </table>
        </div>

        <div class="section-title">Executive Summary</div>
        <div class="summary-cards">
            <div class="card">
                <div class="card-num">{{.TotalTimelineEvents}}</div>
                <div class="card-label">Timeline Events Analyzed</div>
            </div>
            <div class="card critical">
                <div class="card-num">{{len .SigmaAlerts}}</div>
                <div class="card-label">Sigma Detections</div>
            </div>
            <div class="card warning">
                <div class="card-num">{{len .SuspiciousProcesses}}</div>
                <div class="card-label">Suspicious Processes</div>
            </div>
            <div class="card success">
                <div class="card-num">{{len .YaraMatches}}</div>
                <div class="card-label">YARA Memory Matches</div>
            </div>
        </div>

        <div class="section-title">Threat Detections (Sigma Engine)</div>
        {{if .SigmaAlerts}}
        <table class="data-table">
            <thead>
                <tr><th>Severity</th><th>Rule Title</th><th>Log Source</th><th>Matched Context</th></tr>
            </thead>
            <tbody>
                {{range .SigmaAlerts}}
                <tr>
                    <td><span class="badge badge-{{.Level}}">{{.Level}}</span></td>
                    <td><strong>{{.RuleTitle}}</strong></td>
                    <td><code>{{.Path}}</code></td>
                    <td><code>{{.MatchedLog}}</code></td>
                </tr>
                {{end}}
            </tbody>
        </table>
        {{else}}
        <p style="color: #666; font-style: italic;">No Sigma rule triggers recorded.</p>
        {{end}}

        <div class="page-break"></div>

        <div class="section-title">Volatile Memory & Suspicious Processes</div>
        {{if .SuspiciousProcesses}}
        <table class="data-table">
            <thead>
                <tr><th>PID</th><th>Process</th><th>User</th><th>Flag Reason</th><th>Command Line</th></tr>
            </thead>
            <tbody>
                {{range .SuspiciousProcesses}}
                <tr>
                    <td>{{.PID}}</td>
                    <td><strong>{{.Name}}</strong></td>
                    <td>{{.Username}}</td>
                    <td style="color: #C42B1C; font-weight: bold;">{{.FlagReason}}</td>
                    <td><code>{{.CommandLine}}</code></td>
                </tr>
                {{end}}
            </tbody>
        </table>
        {{else}}
        <p style="color: #666; font-style: italic;">No anomalous process executions detected.</p>
        {{end}}

        <div class="section-title">YARA In-Memory Matches</div>
        {{if .YaraMatches}}
        <table class="data-table">
            <thead>
                <tr><th>Rule</th><th>PID</th><th>Process</th><th>Severity</th><th>Matched</th></tr>
            </thead>
            <tbody>
                {{range .YaraMatches}}
                <tr>
                    <td><strong>{{.RuleName}}</strong></td>
                    <td>{{.PID}}</td>
                    <td>{{.ProcessName}}</td>
                    <td><span class="badge badge-{{.Severity}}">{{.Severity}}</span></td>
                    <td><code>{{.MatchedData}}</code></td>
                </tr>
                {{end}}
            </tbody>
        </table>
        {{else}}
        <p style="color: #666; font-style: italic;">No YARA memory signatures matched.</p>
        {{end}}

        <div class="section-title">Timeline Sample (Key Events)</div>
        <table class="data-table">
            <thead>
                <tr><th>Timestamp</th><th>Source</th><th>Type</th><th>Description</th></tr>
            </thead>
            <tbody>
                {{range .TimelineSample}}
                <tr>
                    <td style="white-space: nowrap;">{{.Timestamp}}</td>
                    <td>{{.Source}}</td>
                    <td>{{.EventType}}</td>
                    <td>{{.Description}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>

        <div class="footer">
            Generated automatically by Meteor Forensic Workbench | Confidential Forensic Artifact
        </div>
    </div>
</body>
</html>
`
