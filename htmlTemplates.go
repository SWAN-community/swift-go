/* ****************************************************************************
 * Copyright 2020 51 Degrees Mobile Experts Limited (51degrees.com)
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not
 * use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
 * License for the specific language governing permissions and limitations
 * under the License.
 * ***************************************************************************/

package swift

import (
	"html/template"
	"strings"
)

var bodyStyle = `
body {
	margin: 0;
	padding: 0;
	font-family: nunito, sans-serif;
	font-size: 16px;
	font-weight: 600;
	background-color: {{.BackgroundColor}};
	color: {{.MessageColor}};
	height: 100vh;         
	display: flex;
	justify-content: center;
	align-items: center; }
table {
	text-align: center; }
`

var progressRedirect = `
{{if .Debug}}
<tr>
	<td>
		<style>
			.debug {
				text-align:left;
				font-weight:initial;
			}
			.debug tr td {
				word-wrap:break-word;
				word-break:break-all;
			}
		</style>
		<table class="debug">
			<tr><th>TimeStamp:</th><td>{{.TimeStamp}}</td></tr>
			<tr><th>TimeValid:</th><td>{{.IsTimeStampValid}}</td></tr>
			<tr><th>ReturnUrl:</th><td>{{.ReturnURL}}</td></tr>
			<tr><th>AccessNode:</th><td>{{.AccessNode}}</td></tr>
			<tr><th>HomeNode:</th><td>{{.HomeNode.Domain}}</td></tr>
			<tr><th>NodesVisited:</th><td>{{.NodesVisited}}</td></tr>
			<tr><th>NodeCount:</th><td>{{.NodeCount}}</td></tr>
			<tr><th>NextURL:</th><td>{{.NextURL}}</td></tr>
		</table>
		<table class="debug">
		<tr><th>Key</th><th>Value</th><th>Created</th><th>Expires</th><th>Conflict</th></tr>
		{{range .Values}} 
		<tr><td>{{.Key}}</td><td>{{.Value}}</td><td>{{.Created}}</td><td>{{.Expires}}</td><td>{{.Conflict}}</td></tr>
		{{end}}
		</table>
	</td>
</tr>
<tr><td><a href="{{.NextURL}}">Next</a></td></tr>
{{else}}
<meta http-equiv="refresh" content="0;URL='{{.NextURL}}'"/>
{{end}}`

var progressUI = `
<tr>
	<td>
		<p style="padding-bottom: 2.5em;">{{.Message}}</p>
	</td>
</tr>
<tr>
	<td>
		<div style="display:grid; width: {{.SVGSize}}px; margin: auto; line-height: {{.SVGSize}}px;">
			<style>
				div, svg {
					grid-column: 1;
					grid-row: 1;
				}
			</style>
			<div>{{.PercentageComplete}}%</div>
			<svg style="z-index: -1; stroke:{{.ProgressColor}}; fill:none; stroke-width: {{.SVGStroke}}; width: {{.SVGSize}}px; height: {{.SVGSize}}px;">
				<path d="{{.SVGPath}}"></path>
			</svg>
		</div>
	</td>
</tr>`

var progressTemplate = newHTMLTemplate("progress", `
<!DOCTYPE html>
<html lang="{{.Language}}">
<head>
	<meta charset="utf-8" />
	<title>{{.Title}}</title>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<link rel="icon" href="data:;base64,=">
	<style>`+bodyStyle+`</style>
</head>
<body><table>`+progressUI+progressRedirect+`</table></body>
</html>`)

var blankTemplate = newHTMLTemplate("blank", `
<!DOCTYPE html>
<html lang="{{.Language}}">
<head>
	<meta charset="utf-8" />
	<link rel="icon" href="data:;base64,=">
</head>
<body style="background-color: {{.BackgroundColor}}">
	<meta http-equiv="refresh" content="0;URL='{{.NextURL}}'"/>
</body>
</html>`)

var malformedTemplate = newHTMLTemplate("malformed", `
<!DOCTYPE html>
<html lang="{{.Language}}">
<head>
	<meta charset="utf-8" />
	<title>Bad Request</title>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<link rel="icon" href="data:;base64,=">
	<style>`+bodyStyle+`</style>
</head>
<body>
	<table style="text-align: center; background-color: white; padding: 1em; border: solid black 2px;">
		<tr>
			<td>
				<p>Invalid request.</p>
				<p>Use the settings option in your web browser to disable tracking prevention.</p>
			</td>
		</tr>        
		<tr>
			<td style="padding: 0.5em;">
				<a href="javascript:history.go(-1)" style="display: inline; padding: 0.5em; background-color:black; text-decoration: none; color: white; border: none;">Try Again</a>
			</td>
		</tr>
	</table>
</body>
</html>`)

var registerTemplate = newHTMLTemplate("register", `
<!DOCTYPE html>
<html lang="{{.Language}}">
<head>
	<meta charset="utf-8" />
	<title>Shared Web State - Register Node</title>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<link rel="icon" href="data:;base64,=">
	<style>`+bodyStyle+`</style>
</head>
<body>
	<form action="register" method="GET">
	<table style="text-align: left;">
		<tr>
			<td colspan="3">
				{{if not .ReadOnly}}
				<p>Register node '{{.Domain}}' to a network.</p>
				{{else}}
				<p>Success. Node '{{.Domain}}' registered to network '{{.Network}}'.</p>
				{{end}}
			</td>
		</tr>
		<tr>
			<td>
				<p><label for="network">Network</label></p>
			</td>
			<td>
				<p><input type="text" maxlength="20" id="network" name="network" value="{{.Network}}" {{if .ReadOnly}}disabled{{end}}></p>
			</td>
			<td>
				{{if .DisplayErrors}}
				<p>{{.NetworkError}}</p>
				{{end}}
			</td>
		</tr>
		<tr>
			<td>
				<p><label for="expires">Expires</label></p>
			</td>
			<td>
				<p><input type="date" id="expires" name="expires" value="{{.ExpiresString}}" {{if .ReadOnly}}disabled{{end}}></p>
			</td>
			<td>
				{{if .DisplayErrors}}
				<p>{{.ExpiresError}}</p>
				{{end}}
			</td>
		</tr>
		<tr>
			<td>
				<p><label for="0">Access Node</label></p>
			</td>
			<td>
				<p><input type="radio" id="access" name="role" value="0" {{if .ReadOnly}}disabled{{end}} {{if eq .Role 0}}checked{{end}}></p>
			</td>
			<td rowspan="2">
				{{if .DisplayErrors}}
				<p>{{.RoleError}}</p>
				{{end}}
			</td>
		</tr>
		<tr>
			<td>
				<p><label for="1">Storage Node</label></p>
			</td>
			<td>
				<p><input type="radio" id="storage" name="role" value="1" {{if .ReadOnly}}disabled{{end}} {{if eq .Role 1}}checked{{end}}></p>
			</td>
		</tr>
		<tr>
			<td colspan="3">
				{{if .DisplayErrors}}
				<p>{{.Error}}</p>
				{{end}}
			</td>
		</tr>        
		<tr>
			{{if not .ReadOnly}}
			<td colspan="3" style="text-align: center;">
				<input type="submit">
			</td>
			{{end}}
		</tr>        
	</table>
	</form>
</body>
</html>`)

var warningTemplate = newHTMLTemplate("warning", `
<!DOCTYPE html>
<html lang="{{.Language}}">
<head>
	<meta charset="utf-8" />
	<title>{{.Title}}</title>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<link rel="icon" href="data:;base64,=">
	<style>`+bodyStyle+`</style>
</head>
<body>
	<table style="text-align: center; background-color: white; padding: 1em; border: solid black 2px;">
		<tr>
			<td>
				<p>Cookies are required to access this operation.</p>
				<p>Use the settings option in your web browser to enable cookies.</p>
				<p>You may need to disable tracking prevention.</p>
			</td>
		</tr>
		<tr>
			<td style="padding: 0.5em;">
				<a href="{{.NextURL}}" style="display: inline; padding: 0.5em; background-color:black; text-decoration: none; color: white; border: none;">Try Again</a>
			</td>
		</tr>
		{{if .Debug}}
		<tr>
			<td>
				<style>
					.debug {
						text-align:left;
						font-weight:initial;
					}
					.debug tr td {
						word-wrap:break-word;
						word-break:break-all;
					}
				</style>
				<table class="debug">
					<tr><th>TimeStamp:</th><td>{{.TimeStamp}}</td></tr>
					<tr><th>TimeValid:</th><td>{{.IsTimeStampValid}}</td></tr>
					<tr><th>ReturnUrl:</th><td>{{.ReturnURL}}</td></tr>
					<tr><th>AccessNode:</th><td>{{.AccessNode}}</td></tr>
					<tr><th>HomeNode:</th><td>{{.HomeNode}}</td></tr>
					<tr><th>NodesVisited:</th><td>{{.NodesVisited}}</td></tr>
					<tr><th>NodeCount:</th><td>{{.NodeCount}}</td></tr>
					<tr><th>NextURL:</th><td>{{.NextURL}}</td></tr>
				</table>
				<table class="debug">
				<tr><th>Key</th><th>Value</th><th>Created</th><th>Expires</th><th>Conflict</th></tr>
				{{range .Values}} 
				<tr><td>{{.Key}}</td><td>{{.Value}}</td><td>{{.Created}}</td><td>{{.Expires}}</td><td>{{.Conflict}}</td></tr>
				{{end}}
				</table>
			</td>
		</tr>
		{{end}}
	</table>
</body>
</html>`)

var postMessageTemplate = newHTMLTemplate("postMessage", `
<!DOCTYPE html>
<html lang="{{.Language}}">
<head>
	<meta charset="utf-8" />
	<link rel="icon" href="data:;base64,=">
	<style>`+bodyStyle+`</style>
</head>
<body><table>`+progressUI+`</table>
	<script>
		window.opener.postMessage("{{.Results}}","{{.ReturnURL}}");
	</script>
</body>
</html>`)

var javaScriptProgressTemplate = newJavaScriptTemplate("javaScriptProgress", `
var s=document.createElement("script");
s.src="{{.NextURL}}";
document.currentScript.parentNode.appendChild(s);`)

var javaScriptReturnTemplate = newJavaScriptTemplate("javaScriptReturn", `
var s=document.createElement("script");
s.innerText="{{.Table}}Complete('{{.Results}}')";
document.currentScript.parentNode.appendChild(s);`)

func newHTMLTemplate(n string, h string) *template.Template {
	c := removeHTMLWhiteSpace(h)
	return template.Must(template.New(n).Parse(c))
}

func newJavaScriptTemplate(n string, h string) *template.Template {
	c := removeHTMLWhiteSpace(h)
	return template.Must(template.New(n).Parse(c))
}

// Removes white space from the HTML string provided whilst retaining valid
// HTML.
func removeHTMLWhiteSpace(h string) string {
	var sb strings.Builder
	for i, r := range h {

		// Only write out runes that are not control characters.
		if r != '\r' && r != '\n' && r != '\t' {

			// Only write this rune if the rune is not a space, or if it is a
			// space the preceding rune is not a space.
			if i == 0 || r != ' ' || h[i-1] != ' ' {
				sb.WriteRune(r)
			}
		}
	}
	return sb.String()
}
