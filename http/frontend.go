package http

// this is what happens when a backend dev does frontend.
const frontendTemplate = `
<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Cabourotte</title>
  <style>
    body {
        font-family: serif, Arial;
        font-size: 17px;
    }
    .healthcheck {
        float: left;
        margin: 10px;
        border: 1px solid black;
        padding: 5px;
        height: 250px;
        width: 500px;
    }
    ul {
        list-style: none;
    }
    h3 {
        margin-left: 15px;
    }
    li {
        margin-bottom: 9px;
    }
    .success {
        background-color: #aaff80;
    }
    .failure {
        background-color: #ffb3b3;
    }
  </style>
</head>
<body>
  <h1>Healthchecks</h1>
  {{ range .}}
  <div class="healthcheck {{ if .Success}}success{{else}}failure{{end}}" id="healthcheck-{{ .Name }}">
    <h3>{{ .Name }}</h3>
    <ul>
      <li><b>Summary</b>: {{.Summary }}</li>
      <li><b>Message</b>: {{ .Message }}</li>
      <li><b>Timestamp</b>: {{ .HealthcheckTimestamp }}</li>
      {{ if .Labels }}
      <li><b>Labels</b>: {{ range $key, $value := .Labels }}<b>{{ $key }}</b> = {{ $value }} | {{ end }}
      </li>
      {{ end }}
    </ul>
  </div>
{{ end }}
</body>
</html>



`
