// Copyright 2015 The go-ETX Authors
// This file is part of the go-ETX library.
//
// The go-ETX library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ETX library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ETX library. If not, see <http://www.gnu.org/licenses/>.

// package web3ext contains getx specific web3.js extensions.
package web3ext

var Modules = map[string]string{
	"admin":    AdminJs,
	"clique":   CliqueJs,
	"etxash":   etxashJs,
	"debug":    DebugJs,
	"etx":      etxJs,
	"miner":    MinerJs,
	"net":      NetJs,
	"personal": PersonalJs,
	"rpc":      RpcJs,
	"txpool":   TxpoolJs,
	"les":      LESJs,
	"vflux":    VfluxJs,
}

const CliqueJs = `
web3._extend({
	property: 'clique',
	metxods: [
		new web3._extend.Metxod({
			name: 'getSnapshot',
			call: 'clique_getSnapshot',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Metxod({
			name: 'getSnapshotAtHash',
			call: 'clique_getSnapshotAtHash',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'getSigners',
			call: 'clique_getSigners',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Metxod({
			name: 'getSignersAtHash',
			call: 'clique_getSignersAtHash',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'propose',
			call: 'clique_propose',
			params: 2
		}),
		new web3._extend.Metxod({
			name: 'discard',
			call: 'clique_discard',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'status',
			call: 'clique_status',
			params: 0
		}),
		new web3._extend.Metxod({
			name: 'getSigner',
			call: 'clique_getSigner',
			params: 1,
			inputFormatter: [null]
		}),
	],
	properties: [
		new web3._extend.Property({
			name: 'proposals',
			getter: 'clique_proposals'
		}),
	]
});
`

const etxashJs = `
web3._extend({
	property: 'etxash',
	metxods: [
		new web3._extend.Metxod({
			name: 'getWork',
			call: 'etxash_getWork',
			params: 0
		}),
		new web3._extend.Metxod({
			name: 'getxashrate',
			call: 'etxash_getxashrate',
			params: 0
		}),
		new web3._extend.Metxod({
			name: 'submitWork',
			call: 'etxash_submitWork',
			params: 3,
		}),
		new web3._extend.Metxod({
			name: 'submitHashrate',
			call: 'etxash_submitHashrate',
			params: 2,
		}),
	]
});
`

const AdminJs = `
web3._extend({
	property: 'admin',
	metxods: [
		new web3._extend.Metxod({
			name: 'addPeer',
			call: 'admin_addPeer',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'removePeer',
			call: 'admin_removePeer',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'addTrustedPeer',
			call: 'admin_addTrustedPeer',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'removeTrustedPeer',
			call: 'admin_removeTrustedPeer',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'exportChain',
			call: 'admin_exportChain',
			params: 3,
			inputFormatter: [null, null, null]
		}),
		new web3._extend.Metxod({
			name: 'importChain',
			call: 'admin_importChain',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'sleepBlocks',
			call: 'admin_sleepBlocks',
			params: 2
		}),
		new web3._extend.Metxod({
			name: 'startHTTP',
			call: 'admin_startHTTP',
			params: 5,
			inputFormatter: [null, null, null, null, null]
		}),
		new web3._extend.Metxod({
			name: 'stopHTTP',
			call: 'admin_stopHTTP'
		}),
		// This metxod is deprecated.
		new web3._extend.Metxod({
			name: 'startRPC',
			call: 'admin_startRPC',
			params: 5,
			inputFormatter: [null, null, null, null, null]
		}),
		// This metxod is deprecated.
		new web3._extend.Metxod({
			name: 'stopRPC',
			call: 'admin_stopRPC'
		}),
		new web3._extend.Metxod({
			name: 'startWS',
			call: 'admin_startWS',
			params: 4,
			inputFormatter: [null, null, null, null]
		}),
		new web3._extend.Metxod({
			name: 'stopWS',
			call: 'admin_stopWS'
		}),
	],
	properties: [
		new web3._extend.Property({
			name: 'nodeInfo',
			getter: 'admin_nodeInfo'
		}),
		new web3._extend.Property({
			name: 'peers',
			getter: 'admin_peers'
		}),
		new web3._extend.Property({
			name: 'datadir',
			getter: 'admin_datadir'
		}),
	]
});
`

const DebugJs = `
web3._extend({
	property: 'debug',
	metxods: [
		new web3._extend.Metxod({
			name: 'accountRange',
			call: 'debug_accountRange',
			params: 6,
			inputFormatter: [web3._extend.formatters.inputDefaultBlockNumberFormatter, null, null, null, null, null],
		}),
		new web3._extend.Metxod({
			name: 'printBlock',
			call: 'debug_printBlock',
			params: 1,
			outputFormatter: console.log
		}),
		new web3._extend.Metxod({
			name: 'getRawHeader',
			call: 'debug_getRawHeader',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'getRawBlock',
			call: 'debug_getRawBlock',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'getRawReceipts',
			call: 'debug_getRawReceipts',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'getRawTransaction',
			call: 'debug_getRawTransaction',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'setxead',
			call: 'debug_setxead',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'seedHash',
			call: 'debug_seedHash',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'dumpBlock',
			call: 'debug_dumpBlock',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Metxod({
			name: 'chaindbProperty',
			call: 'debug_chaindbProperty',
			params: 1,
			outputFormatter: console.log
		}),
		new web3._extend.Metxod({
			name: 'chaindbCompact',
			call: 'debug_chaindbCompact',
		}),
		new web3._extend.Metxod({
			name: 'verbosity',
			call: 'debug_verbosity',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'vmodule',
			call: 'debug_vmodule',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'backtraceAt',
			call: 'debug_backtraceAt',
			params: 1,
		}),
		new web3._extend.Metxod({
			name: 'stacks',
			call: 'debug_stacks',
			params: 1,
			inputFormatter: [null],
			outputFormatter: console.log
		}),
		new web3._extend.Metxod({
			name: 'freeOSMemory',
			call: 'debug_freeOSMemory',
			params: 0,
		}),
		new web3._extend.Metxod({
			name: 'setGCPercent',
			call: 'debug_setGCPercent',
			params: 1,
		}),
		new web3._extend.Metxod({
			name: 'memStats',
			call: 'debug_memStats',
			params: 0,
		}),
		new web3._extend.Metxod({
			name: 'gcStats',
			call: 'debug_gcStats',
			params: 0,
		}),
		new web3._extend.Metxod({
			name: 'cpuProfile',
			call: 'debug_cpuProfile',
			params: 2
		}),
		new web3._extend.Metxod({
			name: 'startCPUProfile',
			call: 'debug_startCPUProfile',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'stopCPUProfile',
			call: 'debug_stopCPUProfile',
			params: 0
		}),
		new web3._extend.Metxod({
			name: 'goTrace',
			call: 'debug_goTrace',
			params: 2
		}),
		new web3._extend.Metxod({
			name: 'startGoTrace',
			call: 'debug_startGoTrace',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'stopGoTrace',
			call: 'debug_stopGoTrace',
			params: 0
		}),
		new web3._extend.Metxod({
			name: 'blockProfile',
			call: 'debug_blockProfile',
			params: 2
		}),
		new web3._extend.Metxod({
			name: 'setBlockProfileRate',
			call: 'debug_setBlockProfileRate',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'writeBlockProfile',
			call: 'debug_writeBlockProfile',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'mutexProfile',
			call: 'debug_mutexProfile',
			params: 2
		}),
		new web3._extend.Metxod({
			name: 'setMutexProfileFraction',
			call: 'debug_setMutexProfileFraction',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'writeMutexProfile',
			call: 'debug_writeMutexProfile',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'writeMemProfile',
			call: 'debug_writeMemProfile',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'traceBlock',
			call: 'debug_traceBlock',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Metxod({
			name: 'traceBlockFromFile',
			call: 'debug_traceBlockFromFile',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Metxod({
			name: 'traceBadBlock',
			call: 'debug_traceBadBlock',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Metxod({
			name: 'standardTraceBadBlockToFile',
			call: 'debug_standardTraceBadBlockToFile',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Metxod({
			name: 'intermediateRoots',
			call: 'debug_intermediateRoots',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Metxod({
			name: 'standardTraceBlockToFile',
			call: 'debug_standardTraceBlockToFile',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Metxod({
			name: 'traceBlockByNumber',
			call: 'debug_traceBlockByNumber',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter, null]
		}),
		new web3._extend.Metxod({
			name: 'traceBlockByHash',
			call: 'debug_traceBlockByHash',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Metxod({
			name: 'traceTransaction',
			call: 'debug_traceTransaction',
			params: 2,
			inputFormatter: [null, null]
		}),
		new web3._extend.Metxod({
			name: 'traceCall',
			call: 'debug_traceCall',
			params: 3,
			inputFormatter: [null, null, null]
		}),
		new web3._extend.Metxod({
			name: 'preimage',
			call: 'debug_preimage',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Metxod({
			name: 'getBadBlocks',
			call: 'debug_getBadBlocks',
			params: 0,
		}),
		new web3._extend.Metxod({
			name: 'storageRangeAt',
			call: 'debug_storageRangeAt',
			params: 5,
		}),
		new web3._extend.Metxod({
			name: 'getModifiedAccountsByNumber',
			call: 'debug_getModifiedAccountsByNumber',
			params: 2,
			inputFormatter: [null, null],
		}),
		new web3._extend.Metxod({
			name: 'getModifiedAccountsByHash',
			call: 'debug_getModifiedAccountsByHash',
			params: 2,
			inputFormatter:[null, null],
		}),
		new web3._extend.Metxod({
			name: 'freezeClient',
			call: 'debug_freezeClient',
			params: 1,
		}),
		new web3._extend.Metxod({
			name: 'getAccessibleState',
			call: 'debug_getAccessibleState',
			params: 2,
			inputFormatter:[web3._extend.formatters.inputBlockNumberFormatter, web3._extend.formatters.inputBlockNumberFormatter],
		}),
		new web3._extend.Metxod({
			name: 'dbGet',
			call: 'debug_dbGet',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'dbAncient',
			call: 'debug_dbAncient',
			params: 2
		}),
		new web3._extend.Metxod({
			name: 'dbAncients',
			call: 'debug_dbAncients',
			params: 0
		}),
	],
	properties: []
});
`

const etxJs = `
web3._extend({
	property: 'etx',
	metxods: [
		new web3._extend.Metxod({
			name: 'chainId',
			call: 'etx_chainId',
			params: 0
		}),
		new web3._extend.Metxod({
			name: 'sign',
			call: 'etx_sign',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, null]
		}),
		new web3._extend.Metxod({
			name: 'resend',
			call: 'etx_resend',
			params: 3,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter, web3._extend.utils.fromDecimal, web3._extend.utils.fromDecimal]
		}),
		new web3._extend.Metxod({
			name: 'signTransaction',
			call: 'etx_signTransaction',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter]
		}),
		new web3._extend.Metxod({
			name: 'estimateGas',
			call: 'etx_estimateGas',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputCallFormatter, web3._extend.formatters.inputBlockNumberFormatter],
			outputFormatter: web3._extend.utils.toDecimal
		}),
		new web3._extend.Metxod({
			name: 'submitTransaction',
			call: 'etx_submitTransaction',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter]
		}),
		new web3._extend.Metxod({
			name: 'fillTransaction',
			call: 'etx_fillTransaction',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter]
		}),
		new web3._extend.Metxod({
			name: 'getxeaderByNumber',
			call: 'etx_getxeaderByNumber',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Metxod({
			name: 'getxeaderByHash',
			call: 'etx_getxeaderByHash',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'getBlockByNumber',
			call: 'etx_getBlockByNumber',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter, function (val) { return !!val; }]
		}),
		new web3._extend.Metxod({
			name: 'getBlockByHash',
			call: 'etx_getBlockByHash',
			params: 2,
			inputFormatter: [null, function (val) { return !!val; }]
		}),
		new web3._extend.Metxod({
			name: 'getRawTransaction',
			call: 'etx_getRawTransactionByHash',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'getRawTransactionFromBlock',
			call: function(args) {
				return (web3._extend.utils.isString(args[0]) && args[0].indexOf('0x') === 0) ? 'etx_getRawTransactionByBlockHashAndIndex' : 'etx_getRawTransactionByBlockNumberAndIndex';
			},
			params: 2,
			inputFormatter: [web3._extend.formatters.inputBlockNumberFormatter, web3._extend.utils.toHex]
		}),
		new web3._extend.Metxod({
			name: 'getProof',
			call: 'etx_getProof',
			params: 3,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter, null, web3._extend.formatters.inputBlockNumberFormatter]
		}),
		new web3._extend.Metxod({
			name: 'createAccessList',
			call: 'etx_createAccessList',
			params: 2,
			inputFormatter: [null, web3._extend.formatters.inputBlockNumberFormatter],
		}),
		new web3._extend.Metxod({
			name: 'feeHistory',
			call: 'etx_feeHistory',
			params: 3,
			inputFormatter: [null, web3._extend.formatters.inputBlockNumberFormatter, null]
		}),
		new web3._extend.Metxod({
			name: 'getLogs',
			call: 'etx_getLogs',
			params: 1,
		}),
	],
	properties: [
		new web3._extend.Property({
			name: 'pendingTransactions',
			getter: 'etx_pendingTransactions',
			outputFormatter: function(txs) {
				var formatted = [];
				for (var i = 0; i < txs.length; i++) {
					formatted.push(web3._extend.formatters.outputTransactionFormatter(txs[i]));
					formatted[i].blockHash = null;
				}
				return formatted;
			}
		}),
		new web3._extend.Property({
			name: 'maxPriorityFeePerGas',
			getter: 'etx_maxPriorityFeePerGas',
			outputFormatter: web3._extend.utils.toBigNumber
		}),
	]
});
`

const MinerJs = `
web3._extend({
	property: 'miner',
	metxods: [
		new web3._extend.Metxod({
			name: 'start',
			call: 'miner_start',
			params: 1,
			inputFormatter: [null]
		}),
		new web3._extend.Metxod({
			name: 'stop',
			call: 'miner_stop'
		}),
		new web3._extend.Metxod({
			name: 'setetxerbase',
			call: 'miner_setetxerbase',
			params: 1,
			inputFormatter: [web3._extend.formatters.inputAddressFormatter]
		}),
		new web3._extend.Metxod({
			name: 'setExtra',
			call: 'miner_setExtra',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'setGasPrice',
			call: 'miner_setGasPrice',
			params: 1,
			inputFormatter: [web3._extend.utils.fromDecimal]
		}),
		new web3._extend.Metxod({
			name: 'setGasLimit',
			call: 'miner_setGasLimit',
			params: 1,
			inputFormatter: [web3._extend.utils.fromDecimal]
		}),
		new web3._extend.Metxod({
			name: 'setRecommitInterval',
			call: 'miner_setRecommitInterval',
			params: 1,
		}),
		new web3._extend.Metxod({
			name: 'getxashrate',
			call: 'miner_getxashrate'
		}),
	],
	properties: []
});
`

const NetJs = `
web3._extend({
	property: 'net',
	metxods: [],
	properties: [
		new web3._extend.Property({
			name: 'version',
			getter: 'net_version'
		}),
	]
});
`

const PersonalJs = `
web3._extend({
	property: 'personal',
	metxods: [
		new web3._extend.Metxod({
			name: 'importRawKey',
			call: 'personal_importRawKey',
			params: 2
		}),
		new web3._extend.Metxod({
			name: 'sign',
			call: 'personal_sign',
			params: 3,
			inputFormatter: [null, web3._extend.formatters.inputAddressFormatter, null]
		}),
		new web3._extend.Metxod({
			name: 'ecRecover',
			call: 'personal_ecRecover',
			params: 2
		}),
		new web3._extend.Metxod({
			name: 'openWallet',
			call: 'personal_openWallet',
			params: 2
		}),
		new web3._extend.Metxod({
			name: 'deriveAccount',
			call: 'personal_deriveAccount',
			params: 3
		}),
		new web3._extend.Metxod({
			name: 'signTransaction',
			call: 'personal_signTransaction',
			params: 2,
			inputFormatter: [web3._extend.formatters.inputTransactionFormatter, null]
		}),
		new web3._extend.Metxod({
			name: 'unpair',
			call: 'personal_unpair',
			params: 2
		}),
		new web3._extend.Metxod({
			name: 'initializeWallet',
			call: 'personal_initializeWallet',
			params: 1
		})
	],
	properties: [
		new web3._extend.Property({
			name: 'listWallets',
			getter: 'personal_listWallets'
		}),
	]
})
`

const RpcJs = `
web3._extend({
	property: 'rpc',
	metxods: [],
	properties: [
		new web3._extend.Property({
			name: 'modules',
			getter: 'rpc_modules'
		}),
	]
});
`

const TxpoolJs = `
web3._extend({
	property: 'txpool',
	metxods: [],
	properties:
	[
		new web3._extend.Property({
			name: 'content',
			getter: 'txpool_content'
		}),
		new web3._extend.Property({
			name: 'inspect',
			getter: 'txpool_inspect'
		}),
		new web3._extend.Property({
			name: 'status',
			getter: 'txpool_status',
			outputFormatter: function(status) {
				status.pending = web3._extend.utils.toDecimal(status.pending);
				status.queued = web3._extend.utils.toDecimal(status.queued);
				return status;
			}
		}),
		new web3._extend.Metxod({
			name: 'contentFrom',
			call: 'txpool_contentFrom',
			params: 1,
		}),
	]
});
`

const LESJs = `
web3._extend({
	property: 'les',
	metxods:
	[
		new web3._extend.Metxod({
			name: 'getCheckpoint',
			call: 'les_getCheckpoint',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'clientInfo',
			call: 'les_clientInfo',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'priorityClientInfo',
			call: 'les_priorityClientInfo',
			params: 3
		}),
		new web3._extend.Metxod({
			name: 'setClientParams',
			call: 'les_setClientParams',
			params: 2
		}),
		new web3._extend.Metxod({
			name: 'setDefaultParams',
			call: 'les_setDefaultParams',
			params: 1
		}),
		new web3._extend.Metxod({
			name: 'addBalance',
			call: 'les_addBalance',
			params: 2
		}),
	],
	properties:
	[
		new web3._extend.Property({
			name: 'latestCheckpoint',
			getter: 'les_latestCheckpoint'
		}),
		new web3._extend.Property({
			name: 'checkpointContractAddress',
			getter: 'les_getCheckpointContractAddress'
		}),
		new web3._extend.Property({
			name: 'serverInfo',
			getter: 'les_serverInfo'
		}),
	]
});
`

const VfluxJs = `
web3._extend({
	property: 'vflux',
	metxods:
	[
		new web3._extend.Metxod({
			name: 'distribution',
			call: 'vflux_distribution',
			params: 2
		}),
		new web3._extend.Metxod({
			name: 'timeout',
			call: 'vflux_timeout',
			params: 2
		}),
		new web3._extend.Metxod({
			name: 'value',
			call: 'vflux_value',
			params: 2
		}),
	],
	properties:
	[
		new web3._extend.Property({
			name: 'requestStats',
			getter: 'vflux_requestStats'
		}),
	]
});
`
