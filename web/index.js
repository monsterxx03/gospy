let p_status_color_map = {
    idle: "white",
    running: "green",
    syscall: "blue",
    gcstop: "yellow",
    dead: "red",
};
let svg_ps = d3.select("div#ps")
    .append("svg")
    .attr("width", window.innerWidth);
let drawPs = data => {
    let _default = {
        x: 100,
        y: 50,
        y_margin: 20,
        width: 100,
        height: 150,
        text_padding: 10,
        stroke: "black"
    };
    svg_ps.attr("height", _default.y + data.length * (_default.height + _default.y_margin));
    let g = svg_ps.selectAll("g").data(data).enter().append("g")
        .attr("transform", (d, i) => `translate(${_default.x}, ${_default.y + (_default.height + _default.y_margin) * i})`);
    g.append("rect")
        .attr("stroke", _default.stroke)
        .attr("width", _default.width)
        .attr("height", _default.height)
        .attr("fill", d => p_status_color_map[d.status]);
    g.append("text")
        .attr("x", _default.text_padding)
        .attr("y", _default.height / 2)
        .text(d => `${d.name}(${d.status})`);
};

let load_ps = () => {
    d3.json("/runtime/ps").then(data => {
        drawPs(data);
    }).catch(error => {
        console.log(error);
    })
};

load_ps();
