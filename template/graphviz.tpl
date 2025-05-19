{{template "base/html" .}}

<head>
  {{template "base/head" .}}
</head>

{{template "base/body" .}}
  <div class="d-flex flex-column align-items-center">
    <h3>Last update: {{.Timestamp}}</h3>
    {{.Image}}
  </div>
{{template "base/footer" .}}

</html>
