var varint = require('varint')
//
//
t = {
    'contract': '0620010181004880220001000000000000000000000000000000000000000000000000000000000000000448880001433204810312443300102212443300088190943d124433002032031244330009320312443301108106124433011922124433011881df0412443301203203124433021022124433020881001244330220802050b9d5cd33b835f53649f25be3ba6e6b8271b6d16c0af8aa97cc11761e417feb1244330209320312442243',
    'TMPL_ADDR_IDX': 0,
    'TMPL_APP_ADDRESS': '50b9d5cd33b835f53649f25be3ba6e6b8271b6d16c0af8aa97cc11761e417feb',
    'TMPL_APP_ID': 607,
    'TMPL_EMITTER_ID': '00010000000000000000000000000000000000000000000000000000000000000004',
    'TMPL_SEED_AMT': 1002000
}


t2 = {
    'contract': '062001018101488008677561726469616e48880001433204810312443300102212443300088190943d124433002032031244330009320312443301108106124433011922124433011881df0412443301203203124433021022124433020881001244330220802050b9d5cd33b835f53\
649f25be3ba6e6b8271b6d16c0af8aa97cc11761e417feb1244330209320312442243',
    'TMPL_ADDR_IDX': 1,
    'TMPL_APP_ADDRESS': '50b9d5cd33b835f53649f25be3ba6e6b8271b6d16c0af8aa97cc11761e417feb',
    'TMPL_APP_ID': 607,
    'TMPL_EMITTER_ID': '677561726469616e',
    'TMPL_SEED_AMT': 1002000
}

function properHex(v) {
    if (v < 10)
        return '0' + v.toString(16)
    else
        return v.toString(16)
}

function populate(v) {
    foo = [
        '0620010181', 
        varint.encode(v["TMPL_ADDR_IDX"]).map (n => properHex(n)).join(''),
        '4880', 
        varint.encode(v["TMPL_EMITTER_ID"].length / 2).map (n => properHex(n)).join(''),
        v["TMPL_EMITTER_ID"],
        '488800014332048103124433001022124433000881', 
        varint.encode(v["TMPL_SEED_AMT"]).map (n => properHex(n)).join(''),
        '124433002032031244330009320312443301108106124433011922124433011881', 
        varint.encode(v["TMPL_APP_ID"]).map (n => properHex(n)).join(''),
        '1244330120320312443302102212443302088100124433022080', 
        varint.encode(v["TMPL_APP_ADDRESS"].length/2).map (n => properHex(n)).join(''),
        v["TMPL_APP_ADDRESS"],
        '1244330209320312442243'
    ].join('')
    return foo
}

if (t["contract"] == populate(t)) {
    console.log("omg it works!")
} else {
    console.log("You are weak")
}

if (t2["contract"] == populate(t2)) {
    console.log("omg it works!")
} else {
    console.log("You are weak")
}
