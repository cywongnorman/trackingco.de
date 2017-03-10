const localsync = require('localsync').default
const Emitter = require('tiny-emitter')
const Auth0 = require('auth0-js')

const log = require('./log')

var domain = 'trackingcode.auth0.com'
const clientId = 'bT2dkdr6IzcVgOcTD5dVuG5NLGn1qps6'
const returnTo = location.protocol + '//' + location.host
const redirectTo = returnTo + '/sites'

const auth0 = new Auth0.WebAuth({
  domain: domain,
  clientID: clientId,
  responseType: 'id_token'
})

module.exports.auth0 = {
  logout () {
    module.exports.setToken(null)
  },

  getLoginURL () {
    var state = Math.random().toString()
    return auth0.client.buildAuthorizeUrl({
      redirectUri: redirectTo,
      nonce: state,
      scope: 'openid'
    })
  },

  parseHash (hash, cb) {
    auth0.parseHash(hash, cb)
  },

  getUserInfo (idToken) {
    return window.fetch('https://trackingcode.auth0.com/userinfo', {
      method: 'GET',
      headers: {'Authorization': 'Bearer ' + idToken},
      mode: 'cors'
    })
      .then(r => r.json())
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
    localStorage.setItem('_tcj', token)
    loggedSync.trigger(!!token)
    loggedEmitter.emit(!!token)
  }
}
module.exports.getToken = function getToken () { return localStorage.getItem('_tcj') }
module.exports.onLoggedStateChange = function onLoggedStateChange (cb) {
  cb(!!module.exports.getToken())
  loggedEmitter.on('logged', cb)
}
