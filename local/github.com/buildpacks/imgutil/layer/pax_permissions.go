package layer

// $sddl = (ConvertFrom-SddlString $sddlValue)
// $sddlBytes = [byte[]]::New($sddl.RawDescriptor.BinaryLength)
// $sddl.RawDescriptor.GetBinaryForm($sddlBytes, 0)
// [Convert]::ToBase64String($sddlBytes)

// owner: BUILTIN/Administrators group: BUILTIN/Administrators ($sddlValue="O:BAG:BA")
const AdministratratorOwnerAndGroupSID = "AQAAgBQAAAAkAAAAAAAAAAAAAAABAgAAAAAABSAAAAAgAgAAAQIAAAAAAAUgAAAAIAIAAA=="

// owner: BUILTIN/Users group: BUILTIN/Users ($sddlValue="O:BUG:BU")
const UserOwnerAndGroupSID = "AQAAgBQAAAAkAAAAAAAAAAAAAAABAgAAAAAABSAAAAAhAgAAAQIAAAAAAAUgAAAAIQIAAA=="
