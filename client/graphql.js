const Lokka = require('lokka').Lokka
const Transport = require('lokka-transport-http').Transport

module.exports = new Lokka({transport: new Transport('/_graphql')})
