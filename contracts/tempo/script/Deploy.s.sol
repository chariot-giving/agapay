// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Script, console} from "forge-std/Script.sol";
import {AgapayRegistry} from "../src/AgapayRegistry.sol";
import {AgapayPaymentRouter} from "../src/AgapayPaymentRouter.sol";

contract DeployScript is Script {
    function run() public {
        vm.startBroadcast();

        AgapayRegistry registry = new AgapayRegistry();
        console.log("AgapayRegistry deployed to:", address(registry));

        AgapayPaymentRouter router = new AgapayPaymentRouter();
        console.log("AgapayPaymentRouter deployed to:", address(router));

        vm.stopBroadcast();
    }
}
