// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

// https://github.com/witnet/elliptic-curve-solidity/blob/master/contracts/helpers/TestEllipticCurve.sol

import "../libraries/ecc/EllipticCurve.sol";

/**
 * @title Test Helper for the EllipticCurve library
 * @author Witnet Foundation
 */
contract TestEllipticCurve {
    function invMod(uint256 _x, uint256 _pp) public pure returns (uint256) {
        return EllipticCurve.invMod(_x, _pp);
    }

    function expMod(uint256 _base, uint256 _exp, uint256 _pp) public pure returns (uint256) {
        return EllipticCurve.expMod(_base, _exp, _pp);
    }

    function toAffine(
        uint256 _x,
        uint256 _y,
        uint256 _z,
        uint256 _pp
    ) public pure returns (uint256, uint256) {
        return EllipticCurve.toAffine(_x, _y, _z, _pp);
    }

    function deriveY(
        uint8 _prefix,
        uint256 _x,
        uint256 _aa,
        uint256 _bb,
        uint256 _pp
    ) public pure returns (uint256) {
        return EllipticCurve.deriveY(_prefix, _x, _aa, _bb, _pp);
    }

    function isOnCurve(
        uint256 _x,
        uint256 _y,
        uint256 _aa,
        uint256 _bb,
        uint256 _pp
    ) public pure returns (bool) {
        return EllipticCurve.isOnCurve(_x, _y, _aa, _bb, _pp);
    }

    function ecInv(uint256 _x, uint256 _y, uint256 _pp) public pure returns (uint256, uint256) {
        return EllipticCurve.ecInv(_x, _y, _pp);
    }

    function ecAdd(
        uint256 _x1,
        uint256 _y1,
        uint256 _x2,
        uint256 _y2,
        uint256 _aa,
        uint256 _pp
    ) public pure returns (uint256, uint256) {
        return EllipticCurve.ecAdd(_x1, _y1, _x2, _y2, _aa, _pp);
    }

    function ecSub(
        uint256 _x1,
        uint256 _y1,
        uint256 _x2,
        uint256 _y2,
        uint256 _aa,
        uint256 _pp
    ) public pure returns (uint256, uint256) {
        return EllipticCurve.ecSub(_x1, _y1, _x2, _y2, _aa, _pp);
    }

    function ecMul(
        uint256 _k,
        uint256 _x,
        uint256 _y,
        uint256 _aa,
        uint256 _pp
    ) public pure returns (uint256, uint256) {
        return EllipticCurve.ecMul(_k, _x, _y, _aa, _pp);
    }
}
