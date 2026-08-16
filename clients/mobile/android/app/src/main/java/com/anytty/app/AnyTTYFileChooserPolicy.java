package com.anytty.app;

final class AnyTTYFileChooserPolicy {
    private AnyTTYFileChooserPolicy() {}

    static boolean allowsSingleImage(String pageUrl, String[] acceptTypes, int mode, int singleOpenMode) {
        if (!AnyTTYLocalUrl.isCanonical(pageUrl) || mode != singleOpenMode || acceptTypes == null || acceptTypes.length == 0) {
            return false;
        }
        for (String type : acceptTypes) {
            if (type == null || !type.toLowerCase(java.util.Locale.ROOT).startsWith("image/")) return false;
        }
        return true;
    }
}
