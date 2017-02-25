const randomWord = require('porreta')

module.exports = function (code) {
  return `(function (d, s, c) {
  var x, h, n = Date.now()
  tc = function (p) {
    p = p || 1
    m = s.getItem('_tcx') > n ? s.getItem('_tch') : '${randomWord()}'
    x = new XMLHttpRequest()
    x.addEventListener('load', function () {
      if (x.status == 200) {
        s.setItem('_tch', x.responseText)
        s.setItem('_tcx', n + 14400000)
      }
    })
    x.open('GET', 'https://t.trackingco.de/' + m + '.xml?r=' + d.referrer + '&c=' + c + '&p=' + p)
    x.send()
  }
  tc()
})(document, localStorage, '${code}')`
}
