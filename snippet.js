(function (d, c) {
  tc = function (p) {
    p = p || 1
    var v = d.createElement('img')
    v.src = 'https://t.trackingco.de/header.gif?r=' + d.referrer + '&c=' + c + '&p=' + p
    d.head.appendChild(v)
  }
  tc()
})(document, 'code')
