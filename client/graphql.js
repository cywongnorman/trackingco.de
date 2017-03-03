const Lokka = require('lokka').Lokka
const Transport = require('lokka-transport-http').Transport

const onLoggedStateChange = require('./auth').onLoggedStateChange
const getToken = require('./auth').getToken

// do object modification here:
var headers = {
  'Authorization': getToken()
}
onLoggedStateChange(isLogged => {
  headers['Authorization'] = getToken()
})

let transport = new Transport('/_graphql', {
  headers: headers
})

module.exports = new Lokka({transport})
