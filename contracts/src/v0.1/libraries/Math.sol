// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

/**
 * @dev Math library is a library that contains different mathematical functions for calculations
 *
 */

library Math {
    /**
     * @dev Returns max value given two numbers
     *
     */
    function max(uint256 a, uint256 b) internal pure returns (uint256) {
        if (a > b) {
            return a;
        }
        return b;
    }

    /**
     * @dev Returns the kth value of the ordered array
     * See: http://www.cs.yale.edu/homes/aspnes/pinewiki/QuickSelect.html
     * @param _a The list of elements to pull from
     * @param _k The index, 1 based, of the elements you want to pull from when ordered
     */
    function quickselect(int256[] memory _a, uint256 _k) internal pure returns (int256 pivot) {
        require(_k > 0, "QS01");
        require(_a.length > 0, "QS02");
        require(_k <= _a.length, "QS03");

        int256[] memory a = _a;
        uint256 k = _k;
        uint256 aLen = a.length;
        int256[] memory a1 = new int256[](aLen);
        int256[] memory a2 = new int256[](aLen);
        uint256 a1Len;
        uint256 a2Len;
        uint256 i;

        while (true) {
            pivot = a[aLen / 2];
            a1Len = 0;
            a2Len = 0;
            for (i = 0; i < aLen; i++) {
                if (a[i] < pivot) {
                    a1[a1Len] = a[i];
                    a1Len++;
                } else if (a[i] > pivot) {
                    a2[a2Len] = a[i];
                    a2Len++;
                }
            }
            if (k <= a1Len) {
                aLen = a1Len;
                (a, a1) = swap(a, a1);
            } else if (k > (aLen - a2Len)) {
                k = k - (aLen - a2Len);
                aLen = a2Len;
                (a, a2) = swap(a, a2);
            } else {
                return pivot;
            }
        }
    }

    /**
     * @dev Swaps the pointers to two uint256 arrays in memory
     * @param _a The pointer to the first in memory array
     * @param _b The pointer to the second in memory array
     */
    function swap(int256[] memory _a, int256[] memory _b) private pure returns (int256[] memory, int256[] memory) {
        return (_b, _a);
    }
}
