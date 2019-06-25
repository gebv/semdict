--/*
\connect sduser_db
\set ON_ERROR_STOP on
--*/ 

-- fnProposalAndCommonSenseForProposalAcceptOrReject
create or replace function fnproposalandcommonsenseforproposalacceptorreject(p_sduserid bigint, p_proposalid bigint)
  returns table (commonid bigint
  ,proposalid bigint
  ,senseid bigint
  ,proposalstatus enum_proposalstatus
  ,phrase text
  ,word varchar(512)
  ,phantom bool
  ,deletionproposed bool
  ,ownerid bigint
  ,sdusernickname varchar(128)
  ,languageslug text
  ,commonorproposal varchar(128)
  ,whos varchar(512)
  ,kindofchange varchar(128)
  ,iscommon bool
  ,ismine bool
  ) language plpgsql as $$ 
declare v_commonid bigint;
begin
select vari.commonid from vsense_wide as vari where vari.proposalid = p_proposalid and not vari.phantom into v_commonid;
return query(
  select vari.commonid, vari.proposalid, vari.senseid
    ,vari.proposalstatus
  	,vari.phrase, vari.word, vari.phantom, vari.deletionproposed
    ,vari.ownerid, vari.sdusernickname, vari.languageslug
  	,(explainSenseEssenseVsProposals(p_sduserid,vari.commonid,vari.proposalid,vari.ownerid,false,vari.deletionproposed)).*
    ,(explainCommonAndMine(p_sduserid,vari.commonid,vari.proposalid,vari.ownerid,false)).*
  	from vsense_wide as vari where vari.proposalid = p_proposalid and not vari.phantom
	union all 
  	select s.commonid, s.proposalid, s.senseid
    ,cast('n/a' as enum_proposalstatus)
  	,s.phrase, s.word, s.phantom, false as deletionproposed
    ,cast(0 as bigint) as ownerid, '<common>' as sdusernickname, s.languageslug
  	,(explainSenseEssenseVsProposals(p_sduserid,s.commonid,s.proposalid,s.ownerid,s.phantom,false)).*
    ,(explainCommonAndMine(p_sduserid,s.commonid,s.proposalid,s.ownerid,s.phantom)).*
  	from vsense_wide s where id = v_commonid
	order by iscommon desc); end;
$$;

/* fnAcceptOrRejectSenseProposal merges the proposal into the language
  or rejects it. Arguments:
  p_acceptorreject = 1 for accept, 2 for reject

  Returns common id, or if it was a deletionproposed, 
  special value of -1

  FIXME защищаться от изменения записи другим пользователем во время
  просмотра.
 */
-- Продолжать реализацию слияния смыслов
--  сделать удаление и добавление смысла. Сразу историю?
create or replace function fnacceptorrejectsenseproposal(
    p_sduserid bigint, p_proposalid bigint, p_acceptorreject bigint)
  returns table (r_commonid bigint)
  language plpgsql as $$
  declare v_common_phantom bool;
  declare v_deletionproposed bool;
  declare v_languageid int;
  declare v_common_languageid int;
  declare v_row_count int;
  declare update_count int;
  declare v_commonid bigint;
  declare v_proposalid bigint;
  declare v_have_privilege bool;
  begin
  p_proposalid = coalesce(p_proposalid,0);
  lock table themutex;
  select languageid, phantom, deletionproposed, originid 
    from tsense where id=p_proposalid 
    into v_languageid, v_deletionproposed, v_commonid;
  get diagnostics v_row_count = row_count;
  if v_row_count != 1 then
    raise exception 'invalid p_proposalid'; end if;
  -- Check correctness and privileges
  if v_commonid is not null then
    select languageid, phantom from tsense 
      where id=v_commonid 
      into v_common_languageid, v_common_phantom;
      get diagnostics v_row_count = row_count;
      if v_row_count != 1 then
        raise exception 'invalid proposal (common sense is missing)'; end if; 
      if coalesce(v_languageid,0) <> coalesce(v_common_languageid,0) then
        raise exception 'invalid proposal (language mismatch)'; end if; end if;
  select result from isuserhavelanguageprivilege(p_sduserit
    ,4/*'Accept/decline change requests'*/, v_languageid) into v_have_privilege;
  if not v_have_privilege then
    raise exception 'sorry, you have no right to act on this proposal'; end if;
  -- если отказ, то поменять статус и выйти.
  -- если уже удалено и хотим удалить, то отказываем
  -- если уже удалено и хотим поменять, то восстанавливаем
  -- если добавление, то добавляем
  -- если правка, то правим. 
  return query(select cast(0 as bigint));

/*  p_commonid = coalesce(p_commonid,0);
  if coalesce(p_proposalstatus,'n/a') = 'n/a' then
    raise exception 'proposal status must be not null, not "n/a"'; end if;
  if p_evenifidentical then
    raise exception 'invalid parameter p_evenifidentical'; end if;
  if p_proposalid <> 0 then
    select originid, phantom 
      from tsense where id = p_proposalid 
      into v_commonid, v_phantom;
    if coalesce(v_commonid, 0) <> p_commonid then
      raise exception 'origin mismatch'; end if;
    if exists (select 1 from tsense where 
        id = v_commonid 
        and word = p_word 
        and phrase = p_phrase 
        and phantom = v_phantom) then
    -- nothing differs from the official version, delete our proposal
      delete from tsense where id = p_proposalid;
      return query(select true); return; end if;
    v_proposalid = p_proposalid;
  else -- hence p_proposalid=0
    select ensuresenseproposal(p_sduserid, p_commonid) into v_proposalid; end if;
  
  update tsense set 
    proposalstatus = p_proposalstatus
    ,phrase = p_phrase
    ,word = p_word
    where id = v_proposalid;

  get diagnostics update_count = row_count;
  if update_count != 1 then
    raise exception 'expected to update just one record, which didn''t hapen'; end if;
  return query(select v_proposalid); */ return; end; 
$$;

\echo *** senseProposalAcceptOrReject_fn.sql Done