type TypeKindBadgeProps = {
    kind: string;
    size?: "sm" | "md";
};

const kindDisplayNames: Record<string, string> = {
    string_enum: "String Enum",
    int_enum: "Int Enum",
    object: "Object",
    array: "Array",
    primitive: "Primitive",
    string: "String",
    number: "Number",
    boolean: "Boolean",
    integer: "Integer",
};

export function getKindDisplayName(kind: string): string {
    return kindDisplayNames[kind] || kind.charAt(0).toUpperCase() + kind.slice(1);
}

export function TypeKindBadge({ kind, size = "md" }: TypeKindBadgeProps) {
    const sizeClasses = {
        sm: "text-xs px-2 py-1",
        md: "text-sm px-3 py-1.5",
    }[size];

    return (
        <span
            className={`${sizeClasses} inline-block rounded-lg bg-info-bg text-info-text border-2 border-info-border font-bold`}>
            {getKindDisplayName(kind)}
        </span>
    );
}
