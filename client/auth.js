const localsync = require('localsync').default
const Emitter = require('tiny-emitter')

const log = require('./log')

module.exports.auth = {
  logout () {
    localStorage.removeItem('token')
  },

  tryLogin (cb) {
    let token = location.search.slice(1).split('&')
      .map(kv => kv.split('='))
      .filter(([k, v]) => k === 'token')
      .map(([k, v]) => v)[0]

    window.history.replaceState('', '', '/')

    if (token) {
      localStorage.setItem('token', token)
    }

    return token || localStorage.getItem('token')
  }
}

const loggedEmitter = new Emitter()
const loggedSync = localsync('logged', x => x, (val, old, url) => {
  if (val !== old) {
    log.debug(`another tab at ${url} has changed logged state from ${old} to ${val}.`)
    loggedEmitter.emit('logged', val)
  }
})
setTimeout(() => { loggedSync.start(false) }, 1)

module.exports.setToken = function setToken (token) {
  if (token !== module.exports.getToken()) {
    localStorage.setItem('_tcj', JSON.stringify(token))
    loggedSync.trigger(!!token)
    loggedEmitter.emit('logged', !!token)
  }
}
module.exports.getToken = function getToken () {
  try {
    return JSON.parse(localStorage.getItem('_tcj'))
  } catch (e) {
    return null
  }
}
module.exports.onLoggedStateChange = function onLoggedStateChange (cb) {
  cb(!!module.exports.getToken())
  loggedEmitter.on('logged', cb)
}
