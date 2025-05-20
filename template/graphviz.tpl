{{template "base/html" .}}

<head>
  {{template "base/head" .}}
  <style>
    svg .graph .node:hover ellipse {
      stroke-width: 4;
    }
    svg .graph .edge:hover path {
      stroke-width: 4;
    }
  </style>
</head>

{{template "base/body" .}}
  <div class="d-flex flex-column align-items-center">
    <h3>Last update: {{.Timestamp}}</h3>
    {{.Image}}
  </div>
{{template "base/footer" .}}

</html>
