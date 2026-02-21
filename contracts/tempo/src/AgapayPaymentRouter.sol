// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

/// @title ITIP20
/// @notice Minimal interface for TIP-20 tokens (ERC-20 superset with memo support).
interface ITIP20 {
    function transfer(address to, uint256 amount) external returns (bool);
    function transferWithMemo(address to, uint256 amount, bytes32 memo) external;
    function transferFrom(address from, address to, uint256 amount) external returns (bool);
    function balanceOf(address account) external view returns (uint256);
}

/// @title AgapayPaymentRouter
/// @notice Routes Agapay stablecoin payments through TIP-20 transferWithMemo
/// while emitting a richer event that includes the full public header for
/// off-chain indexing. The 32-byte TIP-20 memo carries the IPFS CID of the
/// public header as an on-chain pointer.
///
/// Data durability strategy:
///   1. IPFS stores the full public header (content-addressed, persistent)
///   2. TIP-20 memo stores the CID hash (on-chain, in event log)
///   3. AgapayPayment event carries the full header for convenient indexing
contract AgapayPaymentRouter {
    // -----------------------------------------------------------------------
    // Events
    // -----------------------------------------------------------------------

    /// @notice Emitted for every Agapay payment processed through this router.
    event AgapayPayment(
        address indexed token,
        address indexed from,
        address indexed to,
        uint256 amount,
        bytes32 memo,
        bytes publicHeader
    );

    // -----------------------------------------------------------------------
    // Errors
    // -----------------------------------------------------------------------

    error EmptyRecipient();
    error ZeroAmount();
    error EmptyPublicHeader();

    // -----------------------------------------------------------------------
    // Payment
    // -----------------------------------------------------------------------

    /// @notice Send a TIP-20 payment with the full Agapay public header.
    ///
    /// The caller must have approved this contract to spend `amount` of `token`,
    /// OR call this through a Tempo Transaction batch where the approval and
    /// this call are in the same atomic transaction.
    ///
    /// @param token       The TIP-20 token address to transfer
    /// @param to          The recipient address
    /// @param amount      The amount in token minor units (6 decimals)
    /// @param memo        The 32-byte memo (typically keccak256 of the IPFS CID)
    /// @param publicHeader The full JSON public header bytes for event indexing
    function sendPayment(address token, address to, uint256 amount, bytes32 memo, bytes calldata publicHeader)
        external
    {
        if (to == address(0)) revert EmptyRecipient();
        if (amount == 0) revert ZeroAmount();
        if (publicHeader.length == 0) revert EmptyPublicHeader();

        bool ok = ITIP20(token).transferFrom(msg.sender, address(this), amount);
        require(ok, "AgapayPaymentRouter: transferFrom failed");
        ITIP20(token).transferWithMemo(to, amount, memo);

        emit AgapayPayment(token, msg.sender, to, amount, memo, publicHeader);
    }

    /// @notice Convenience method: the caller handles the TIP-20 transfer
    /// directly and calls this to emit the Agapay event for indexing.
    function emitPaymentEvent(address token, address to, uint256 amount, bytes32 memo, bytes calldata publicHeader)
        external
    {
        if (publicHeader.length == 0) revert EmptyPublicHeader();

        emit AgapayPayment(token, msg.sender, to, amount, memo, publicHeader);
    }
}
