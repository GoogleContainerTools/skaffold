function updateState() {
    $.getJSON("/v1/state", function (state) {
        stateText = JSON.stringify(state, null, 2)
        ports = ""
        for (p in state.forwardedPorts) {
            fpe = state.forwardedPorts[p]
            ports += `<p><a href="http://localhost:${fpe.localPort}">${fpe.resourceType}/${fpe.resourceName}</a></p>`
        }
        $("#pfes").html(ports)
        $("#state").html(`<pre>${stateText}</pre>`)
    });
}

function initControlAPIButtons() {
    $("#checkboxBuild").prop("disabled", autoBuild)
    $("#checkboxSync").prop("disabled", autoSync)
    $("#checkboxDeploy").prop("disabled", autoDeploy)
    $("#trigger").prop("disabled", autoBuild || autoSync || autoDeploy)
}

function onTrigger(evt) {
    buildIntent = $("#checkboxBuild").prop("checked")
    syncIntent = $("#checkboxSync").prop("checked")
    deployIntent = $("#checkboxDeploy").prop("checked")
    let message = JSON.stringify({"build": buildIntent, "sync": syncIntent, "deploy": deployIntent});
    $.post( "/v1/execute", message, function(resp ) {
        alert('successful trigger:\n' + message + '\nresult:\n' + JSON.stringify(resp, null, 2))
    }).fail(function(resp){
        alert('failed trigger:\n' + message + '\nresult:\n' + JSON.stringify(resp, null, 2))
    });

}

$(document).ready(function () {
    oboe("/v1/events")
        .node('result', function (e) {
            console.log(e);
            eventText = JSON.stringify(e.event, null, 2)
            $("#events").prepend(`<tr><td>${e.timestamp}</td><td>${e.entry}</td><td><pre>${eventText}</pre></td></tr>`);
            updateState()
        })
        .done(function (data) {
            console.log(data);
        });

    initControlAPIButtons()
});
