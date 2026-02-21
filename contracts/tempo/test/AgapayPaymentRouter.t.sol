// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Test} from "forge-std/Test.sol";
import {AgapayPaymentRouter, ITIP20} from "../src/AgapayPaymentRouter.sol";

/// @dev Mock TIP-20 token for testing the payment router.
contract MockTIP20 is ITIP20 {
    mapping(address => uint256) public balanceOf;
    mapping(address => mapping(address => uint256)) public allowance;

    bytes32 public lastMemo;

    function mint(address to, uint256 amount) external {
        balanceOf[to] += amount;
    }

    function transfer(address to, uint256 amount) external returns (bool) {
        balanceOf[msg.sender] -= amount;
        balanceOf[to] += amount;
        return true;
    }

    function transferWithMemo(address to, uint256 amount, bytes32 memo) external {
        balanceOf[msg.sender] -= amount;
        balanceOf[to] += amount;
        lastMemo = memo;
    }

    function transferFrom(address from, address to, uint256 amount) external returns (bool) {
        allowance[from][msg.sender] -= amount;
        balanceOf[from] -= amount;
        balanceOf[to] += amount;
        return true;
    }

    function approve(address spender, uint256 amount) external returns (bool) {
        allowance[msg.sender][spender] = amount;
        return true;
    }
}

contract AgapayPaymentRouterTest is Test {
    AgapayPaymentRouter public router;
    MockTIP20 public token;

    address payer = makeAddr("payer");
    address recipient = makeAddr("recipient");

    bytes constant PUBLIC_HEADER = '{"version":"1.0","message_id":"msg_test","recipient_ein":"530196605"}';
    bytes32 constant MEMO = keccak256("bafy1234test");
    uint256 constant AMOUNT = 500_000_000; // $500 USDC

    function setUp() public {
        router = new AgapayPaymentRouter();
        token = new MockTIP20();

        token.mint(payer, 1_000_000_000);

        vm.prank(payer);
        token.approve(address(router), type(uint256).max);
    }

    // -----------------------------------------------------------------------
    // sendPayment
    // -----------------------------------------------------------------------

    function test_SendPayment() public {
        vm.prank(payer);
        router.sendPayment(address(token), recipient, AMOUNT, MEMO, PUBLIC_HEADER);

        assertEq(token.balanceOf(recipient), AMOUNT);
        assertEq(token.balanceOf(payer), 500_000_000);
        assertEq(token.lastMemo(), MEMO);
    }

    function test_SendPayment_EmitsEvent() public {
        vm.expectEmit(true, true, true, true);
        emit AgapayPaymentRouter.AgapayPayment(address(token), payer, recipient, AMOUNT, MEMO, PUBLIC_HEADER);

        vm.prank(payer);
        router.sendPayment(address(token), recipient, AMOUNT, MEMO, PUBLIC_HEADER);
    }

    function test_SendPayment_RevertsOnEmptyRecipient() public {
        vm.prank(payer);
        vm.expectRevert(AgapayPaymentRouter.EmptyRecipient.selector);
        router.sendPayment(address(token), address(0), AMOUNT, MEMO, PUBLIC_HEADER);
    }

    function test_SendPayment_RevertsOnZeroAmount() public {
        vm.prank(payer);
        vm.expectRevert(AgapayPaymentRouter.ZeroAmount.selector);
        router.sendPayment(address(token), recipient, 0, MEMO, PUBLIC_HEADER);
    }

    function test_SendPayment_RevertsOnEmptyHeader() public {
        vm.prank(payer);
        vm.expectRevert(AgapayPaymentRouter.EmptyPublicHeader.selector);
        router.sendPayment(address(token), recipient, AMOUNT, MEMO, "");
    }

    // -----------------------------------------------------------------------
    // emitPaymentEvent
    // -----------------------------------------------------------------------

    function test_EmitPaymentEvent() public {
        vm.expectEmit(true, true, true, true);
        emit AgapayPaymentRouter.AgapayPayment(address(token), payer, recipient, AMOUNT, MEMO, PUBLIC_HEADER);

        vm.prank(payer);
        router.emitPaymentEvent(address(token), recipient, AMOUNT, MEMO, PUBLIC_HEADER);
    }

    function test_EmitPaymentEvent_RevertsOnEmptyHeader() public {
        vm.prank(payer);
        vm.expectRevert(AgapayPaymentRouter.EmptyPublicHeader.selector);
        router.emitPaymentEvent(address(token), recipient, AMOUNT, MEMO, "");
    }

    // -----------------------------------------------------------------------
    // Fuzz
    // -----------------------------------------------------------------------

    function testFuzz_SendPayment_AnyAmount(uint256 amount) public {
        amount = bound(amount, 1, 1_000_000_000);

        token.mint(payer, amount);
        vm.prank(payer);
        token.approve(address(router), amount);

        vm.prank(payer);
        router.sendPayment(address(token), recipient, amount, MEMO, PUBLIC_HEADER);

        assertEq(token.balanceOf(recipient), amount);
    }
}
