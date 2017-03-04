const Auth0 = window.auth0
const localsync = require('localsync').default
const Emitter = require('tiny-emitter')

let auth0options = {
  domain: 'trackingcode.auth0.com',
  clientID: 'bT2dkdr6IzcVgOcTD5dVuG5NLGn1qps6',
  responseType: 'id_token',
  redirectUri: location.protocol + '//' + location.host + '/sites'
}

module.exports.auth0 = new Auth0.WebAuth(auth0options)

const auth0AuthAPI = new Auth0.Authentication(auth0options)
module.exports.getLogoutURL = function () {
  return auth0AuthAPI.buildLogoutUrl({
    returnTo: location.protocol + '//' + location.host
  })
}

const loggedEmitter = new Emitter()
const loggedSync = localsync('logged', x => x, (val, old, url) => {
  if (val !== old) {
    console.log(`another tab at ${url} has changed logged state from ${old} to ${val}.`)
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
